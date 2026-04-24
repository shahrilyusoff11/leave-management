package services_test

import (
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

func TestNegativeBalanceWorkflowRouting(t *testing.T) {
	db, err := gorm.Open(sqlite.Open("file:workflow_negative?mode=memory&cache=shared"), &gorm.Config{})
	require.NoError(t, err)

	err = db.AutoMigrate(
		&models.Role{},
		&models.User{},
		&models.LeaveRequest{},
		&models.LeaveBalance{},
		&models.LeaveTypeConfig{},
		&models.LeaveWorkflow{},
		&models.WorkflowStep{},
		&models.LeaveRequestWorkflowState{},
		&models.Chronology{},
		&models.Department{},
		&models.BlackoutDate{},
		&models.PublicHoliday{},
	)
	require.NoError(t, err)

	zapLogger := logger.NewLogger("test")
	auditLogger := logger.NewAuditLogger(zapLogger)
	holidaySvc := services.NewHolidayService(db)
	deptSvc := services.NewDepartmentService(db)
	workflowSvc := services.NewWorkflowService(db, deptSvc, nil)
	leaveTypeConfigSvc := services.NewLeaveTypeConfigService(db)
	calculator := services.NewLeaveCalculator(holidaySvc, leaveTypeConfigSvc)
	leaveSvc := services.NewLeaveService(db, calculator, auditLogger, nil, holidaySvc, leaveTypeConfigSvc, deptSvc, workflowSvc)

	// Mock today as a working day (it doesn't matter much as long as we have 0 holidays to subtract)
	db.Create(&models.PublicHoliday{
		ID:   uuid.New(),
		Name: "Test Holiday",
		Date: time.Now().AddDate(1, 0, 0), // Far in the future
	})

	pastDate := time.Now().AddDate(-2, 0, 0)
	manager := models.User{ID: uuid.New(), Email: "mgr@test.com", Role: models.RoleManager, IsActive: true, IsConfirmed: true, CreatedAt: pastDate}
	hr := models.User{ID: uuid.New(), Email: "hr@test.com", Role: models.RoleHR, IsActive: true, IsConfirmed: true, CreatedAt: pastDate}
	employee := models.User{ID: uuid.New(), Email: "emp@test.com", Role: models.RoleStaff, ManagerID: &manager.ID, IsActive: true, IsConfirmed: true, CreatedAt: pastDate}

	require.NoError(t, db.Create(&manager).Error)
	require.NoError(t, db.Create(&hr).Error)
	require.NoError(t, db.Create(&employee).Error)

	config := models.LeaveTypeConfig{
		ID:                   uuid.New(),
		LeaveType:            models.LeaveTypeAnnual,
		AllowNegativeBalance: true,
		IsActive:             true,
	}
	require.NoError(t, db.Create(&config).Error)

	// User has 2 balance
	balance := models.LeaveBalance{
		ID:               uuid.New(),
		UserID:           employee.ID,
		LeaveType:        models.LeaveTypeAnnual,
		Year:             time.Now().Year(),
		TotalEntitlement: 2,
		Used:             0,
		IsOverridden:     true,
	}
	require.NoError(t, db.Create(&balance).Error)

	workflowID := uuid.New()
	mgrStepID := uuid.New()
	hrStepID := uuid.New()
	approvedStepID := uuid.New()

	workflow := models.LeaveWorkflow{
		ID:          workflowID,
		LeaveType:   models.LeaveTypeAnnual,
		IsActive:    true,
		FirstStepID: &mgrStepID,
		Steps: []models.WorkflowStep{
			{
				ID:                mgrStepID,
				WorkflowID:        workflowID,
				StepName:          "Manager Approval",
				ResponsibleRole:   models.RoleManager,
				NextStepOnApprove: &hrStepID,
			},
			{
				ID:                hrStepID,
				WorkflowID:        workflowID,
				StepName:          "HR Review (Negative Balance)",
				ResponsibleRole:   models.RoleHR,
				Conditions:        models.JSONMap{"requires_hr_if_negative": true},
				NextStepOnApprove: &approvedStepID,
			},
			{
				ID:              approvedStepID,
				WorkflowID:      workflowID,
				StepName:        "Approved",
				ResponsibleRole: models.RoleHR,
				IsTerminal:      true,
				TerminalStatus:  models.StatusApproved,
			},
		},
	}
	require.NoError(t, db.Create(&workflow).Error)

	// Test 1: Positive Request (Applies 1 day -> Available 2)
	// Should skip HR
	req1 := &models.LeaveRequest{
		LeaveType:    models.LeaveTypeAnnual,
		StartDate:    time.Now().AddDate(0, 0, 1),
		EndDate:      time.Now().AddDate(0, 0, 1),
		DurationDays: 1,
	}
	require.NoError(t, leaveSvc.CreateLeaveRequest(employee.ID, req1))

	var state1 models.LeaveRequestWorkflowState
	db.Preload("CurrentStep").First(&state1, "leave_request_id = ?", req1.ID)
	assert.Equal(t, "Manager Approval", state1.CurrentStep.StepName)

	_, err = leaveSvc.ProcessWorkflowAction(req1.ID, manager.ID, models.StepActionApproved, "Mgr App")
	require.NoError(t, err)

	db.Preload("CurrentStep").First(&state1, "leave_request_id = ?", req1.ID)
	assert.True(t, state1.IsComplete)
	assert.Equal(t, models.StatusApproved, state1.FinalStatus)

	// Test 2: Negative Request (Applies 4 days -> Available 1)
	req2 := &models.LeaveRequest{
		LeaveType:    models.LeaveTypeAnnual,
		StartDate:    time.Now().AddDate(0, 0, 2),
		EndDate:      time.Now().AddDate(0, 0, 5),
		DurationDays: 4,
	}
	require.NoError(t, leaveSvc.CreateLeaveRequest(employee.ID, req2))

	var state2 models.LeaveRequestWorkflowState
	db.Preload("CurrentStep").First(&state2, "leave_request_id = ?", req2.ID)
	assert.Equal(t, "Manager Approval", state2.CurrentStep.StepName)

	_, err = leaveSvc.ProcessWorkflowAction(req2.ID, manager.ID, models.StepActionApproved, "Mgr App")
	require.NoError(t, err)

	db.Preload("CurrentStep").First(&state2, "leave_request_id = ?", req2.ID)
	assert.False(t, state2.IsComplete)
	assert.Equal(t, "HR Review (Negative Balance)", state2.CurrentStep.StepName)
}
