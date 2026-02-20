package services_test

import (
	"fmt"
	"testing"
	"time"

	"leave-management-system/internal/models"
	"leave-management-system/internal/services"
	"leave-management-system/pkg/logger"

	"github.com/glebarez/sqlite"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

// TestFullWorkflowCycle verifies the entire lifecycle of a leave request
func TestFullWorkflowCycle(t *testing.T) {
	fmt.Println("DEBUG: Starting TestFullWorkflowCycle")

	// --- 1. Setup Database & Services ---
	db, err := gorm.Open(sqlite.Open("file::memory:?cache=shared"), &gorm.Config{})
	require.NoError(t, err)

	// Migrate models
	err = db.AutoMigrate(
		&models.User{},
		&models.LeaveRequest{},
		&models.LeaveBalance{},
		&models.LeaveTypeConfig{},
		&models.LeaveWorkflow{},
		&models.WorkflowStep{},
		&models.LeaveRequestWorkflowState{},
		&models.Chronology{},
		&models.UserDelegation{},
		&models.Department{},
		&models.HODDelegation{},
		&models.PublicHoliday{},
		&models.BlackoutDate{},
	)
	require.NoError(t, err)

	fmt.Println("DEBUG: DB Migrated")

	// Initialize Services
	zapLogger := logger.NewLogger("test")
	auditLogger := logger.NewAuditLogger(zapLogger)

	deptSvc := services.NewDepartmentService(db)
	delegationSvc := services.NewDelegationService(db)
	workflowSvc := services.NewWorkflowService(db, deptSvc, delegationSvc)
	leaveTypeConfigSvc := services.NewLeaveTypeConfigService(db)
	holidaySvc := services.NewHolidayService(db)
	calculator := services.NewLeaveCalculator(holidaySvc, leaveTypeConfigSvc)

	// LeaveService needs many dependencies
	leaveSvc := services.NewLeaveService(
		db,
		calculator,
		auditLogger,
		nil, // emailService
		holidaySvc,
		leaveTypeConfigSvc,
		deptSvc,
		workflowSvc,
	)

	fmt.Println("DEBUG: Services Initialized")

	// --- 2. Setup Data ---

	// Create Users
	director := models.User{
		ID:           uuid.New(),
		Email:        "director@example.com",
		PasswordHash: "password_hash",
		Role:         models.RoleHOD,
		FirstName:    "Director",
		LastName:     "One",
		IsActive:     true,
		JoinedDate:   time.Now().AddDate(-5, 0, 0),
	}
	manager := models.User{
		ID:           uuid.New(),
		Email:        "manager@example.com",
		PasswordHash: "password_hash",
		Role:         models.RoleManager,
		FirstName:    "Manager",
		LastName:     "Two",
		ManagerID:    &director.ID,
		IsActive:     true,
		JoinedDate:   time.Now().AddDate(-3, 0, 0),
	}
	employee := models.User{
		ID:           uuid.New(),
		Email:        "employee@example.com",
		PasswordHash: "password_hash",
		Role:         models.RoleStaff,
		FirstName:    "Employee",
		LastName:     "Three",
		ManagerID:    &manager.ID,
		IsActive:     true,
		IsConfirmed:  true,
		JoinedDate:   time.Now().AddDate(-1, 0, 0),
	}

	require.NoError(t, db.Create(&director).Error)
	require.NoError(t, db.Create(&manager).Error)
	require.NoError(t, db.Create(&employee).Error)

	fmt.Println("DEBUG: Users Created")

	// Setup Leave Balance for Employee (10 days)
	initialBalance := models.LeaveBalance{
		ID:               uuid.New(),
		UserID:           employee.ID,
		LeaveType:        models.LeaveTypeAnnual,
		Year:             time.Now().Year(),
		TotalEntitlement: 10,
		Used:             0,
		CreatedAt:        time.Now(),
		UpdatedAt:        time.Now(),
	}
	require.NoError(t, db.Create(&initialBalance).Error)

	// Setup Leave Type Config
	config := models.LeaveTypeConfig{
		ID:        uuid.New(),
		LeaveType: models.LeaveTypeAnnual,
		IsActive:  true,
	}
	require.NoError(t, db.Create(&config).Error)

	// Setup Workflow
	workflowID := uuid.New()
	step1ID := uuid.New()
	step2ID := uuid.New()

	workflow := models.LeaveWorkflow{
		ID:          workflowID,
		LeaveType:   models.LeaveTypeAnnual,
		Version:     1,
		IsActive:    true,
		FirstStepID: &step1ID,
		Steps: []models.WorkflowStep{
			{
				ID:                step1ID,
				WorkflowID:        workflowID,
				StepName:          "Manager Approval",
				ResponsibleRole:   models.RoleManager,
				NextStepOnApprove: &step2ID,
				NextStepOnReject:  nil, // Terminal Reject
				IsTerminal:        false,
			},
			{
				ID:                step2ID,
				WorkflowID:        workflowID,
				StepName:          "Director Approval",
				ResponsibleRole:   models.RoleHOD,
				NextStepOnApprove: nil,
				IsTerminal:        true,
				TerminalStatus:    models.StatusApproved,
			},
		},
	}
	require.NoError(t, db.Create(&workflow).Error)

	fmt.Println("DEBUG: Workflow Configured")

	// --- 3. Execute Verification ---

	// Step 3.1: Submit Request
	now := time.Now()
	daysUntilMonday := (8 - int(now.Weekday())) % 7
	if daysUntilMonday == 0 {
		daysUntilMonday = 7
	}
	nextMonday := now.AddDate(0, 0, daysUntilMonday)

	req := &models.LeaveRequest{
		LeaveType: models.LeaveTypeAnnual,
		StartDate: nextMonday,
		EndDate:   nextMonday.AddDate(0, 0, 2), // Mon, Tue, Wed (3 days)
		Reason:    "Vacation",
	}

	err = leaveSvc.CreateLeaveRequest(employee.ID, req)
	require.NoError(t, err)

	fmt.Println("DEBUG: Leave Request Created")

	// Verify Initial State
	var savedReq models.LeaveRequest
	err = db.Preload("WorkflowState.CurrentStep").First(&savedReq, "id = ?", req.ID).Error
	require.NoError(t, err)
	assert.Equal(t, models.StatusPending, savedReq.Status)
	assert.NotNil(t, savedReq.WorkflowState)
	assert.Equal(t, "Manager Approval", savedReq.WorkflowState.CurrentStep.StepName)

	// Step 3.2: Manager Approves
	// Verify responsible users
	responsibles, err := workflowSvc.GetResponsibleUsers(savedReq.WorkflowState)
	require.NoError(t, err)
	assert.Len(t, responsibles, 1)
	assert.Equal(t, manager.ID, responsibles[0].ID)

	// 4. Manager approves
	_, err = leaveSvc.ProcessWorkflowAction(req.ID, manager.ID, models.StepActionApproved, "Approved by Manager")
	assert.NoError(t, err)

	var state models.LeaveRequestWorkflowState
	err = db.Preload("CurrentStep").First(&state, "leave_request_id = ?", req.ID).Error

	fmt.Println("DEBUG: Manager Approved")

	// Verify Transition
	err = db.Preload("WorkflowState.CurrentStep").First(&savedReq, "id = ?", req.ID).Error
	require.NoError(t, err)
	assert.Equal(t, models.StatusPending, savedReq.Status) // Still pending
	assert.Equal(t, "Director Approval", savedReq.WorkflowState.CurrentStep.StepName)

	// 6. Director approves
	_, err = leaveSvc.ProcessWorkflowAction(req.ID, director.ID, models.StepActionApproved, "Approved by Director")
	assert.NoError(t, err)

	err = db.First(&state, "leave_request_id = ?", req.ID).Error

	fmt.Println("DEBUG: Director Approved")

	// Verify Final State
	err = db.Preload("WorkflowState").First(&savedReq, "id = ?", req.ID).Error
	require.NoError(t, err)
	assert.Equal(t, models.StatusApproved, savedReq.Status)
	assert.True(t, savedReq.WorkflowState.IsComplete)

	// --- 4. Verify Balance Deduction ---
	var finalBalance models.LeaveBalance
	err = db.Where("user_id = ?", employee.ID).First(&finalBalance).Error
	require.NoError(t, err)

	fmt.Println("DEBUG: Checking Balance Deduction")

	expectedUsed := savedReq.DurationDays
	assert.Equal(t, expectedUsed, finalBalance.Used, "Balance used should match request duration")
	assert.Equal(t, 10.0, finalBalance.TotalEntitlement)
	assert.True(t, finalBalance.Used > 0, "Used balance should be greater than 0")

	fmt.Println("DEBUG: Test Completed Successfully")
}
