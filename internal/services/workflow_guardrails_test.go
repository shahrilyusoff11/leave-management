package services_test

import (
	"strings"
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

func TestWorkflowActionEnforcesDocumentConditions(t *testing.T) {
	db, err := gorm.Open(sqlite.Open("file:workflow_guardrails_conditions?mode=memory&cache=shared"), &gorm.Config{})
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
		&models.PublicHoliday{},
		&models.BlackoutDate{},
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

	employee := models.User{
		ID:           uuid.New(),
		Email:        "emp-doc@test.com",
		PasswordHash: "password_hash",
		Role:         models.RoleStaff,
		FirstName:    "Doc",
		LastName:     "Employee",
		IsActive:     true,
		IsConfirmed:  true,
		JoinedDate:   time.Now().AddDate(-1, 0, 0),
	}
	hr := models.User{
		ID:           uuid.New(),
		Email:        "hr-doc@test.com",
		PasswordHash: "password_hash",
		Role:         models.RoleHR,
		FirstName:    "HR",
		LastName:     "Reviewer",
		IsActive:     true,
		IsConfirmed:  true,
		JoinedDate:   time.Now().AddDate(-2, 0, 0),
	}
	require.NoError(t, db.Create(&employee).Error)
	require.NoError(t, db.Create(&hr).Error)

	require.NoError(t, db.Create(&models.LeaveTypeConfig{
		ID:        uuid.New(),
		LeaveType: models.LeaveTypeSick,
		IsActive:  true,
	}).Error)

	require.NoError(t, db.Create(&models.LeaveBalance{
		ID:               uuid.New(),
		UserID:           employee.ID,
		LeaveType:        models.LeaveTypeSick,
		Year:             time.Now().Year(),
		TotalEntitlement: 14,
		IsOverridden:     true,
	}).Error)

	workflowID := uuid.New()
	hrStepID := uuid.New()
	approvedStepID := uuid.New()
	workflow := models.LeaveWorkflow{
		ID:           workflowID,
		LeaveType:    models.LeaveTypeSick,
		WorkflowName: "Sick Leave Approval",
		Version:      1,
		IsActive:     true,
		FirstStepID:  &hrStepID,
		Steps: []models.WorkflowStep{
			{
				ID:                hrStepID,
				WorkflowID:        workflowID,
				StepOrder:         1,
				StepName:          "hr_review",
				StepLabel:         "HR Review",
				ResponsibleRole:   models.RoleHR,
				ActionType:        models.ActionReview,
				RequiresDocument:  true,
				DocumentType:      "medical_certificate",
				Conditions:        models.JSONMap{"requires_document": true, "document_type": "medical_certificate"},
				NextStepOnApprove: &approvedStepID,
			},
			{
				ID:             approvedStepID,
				WorkflowID:     workflowID,
				StepOrder:      2,
				StepName:       "approved",
				ResponsibleRole: models.RoleHR,
				ActionType:     models.ActionApprove,
				IsTerminal:     true,
				TerminalStatus: models.StatusApproved,
			},
		},
	}
	require.NoError(t, db.Create(&workflow).Error)

	req := &models.LeaveRequest{
		LeaveType: models.LeaveTypeSick,
		StartDate: time.Now().AddDate(0, 0, 1),
		EndDate:   time.Now().AddDate(0, 0, 1),
		Reason:    "Medical leave",
	}
	require.NoError(t, leaveSvc.CreateLeaveRequest(employee.ID, req))

	_, err = leaveSvc.ProcessWorkflowAction(req.ID, hr.ID, models.StepActionApproved, "Approved without document")
	require.Error(t, err)
	assert.True(t, strings.Contains(err.Error(), "Missing required medical_certificate"))
}

func TestCreateWorkflowStepRemapsLinksOnClone(t *testing.T) {
	db, err := gorm.Open(sqlite.Open("file:workflow_guardrails_versioning?mode=memory&cache=shared"), &gorm.Config{})
	require.NoError(t, err)

	err = db.AutoMigrate(
		&models.Role{},
		&models.User{},
		&models.LeaveWorkflow{},
		&models.WorkflowStep{},
		&models.LeaveRequest{},
		&models.LeaveRequestWorkflowState{},
		&models.Notification{},
	)
	require.NoError(t, err)

	workflowSvc := services.NewWorkflowService(db, nil, nil)

	workflowID := uuid.New()
	step1ID := uuid.New()
	step2ID := uuid.New()
	workflow := models.LeaveWorkflow{
		ID:           workflowID,
		LeaveType:    models.LeaveTypeAnnual,
		WorkflowName: "Standard",
		Version:      1,
		IsActive:     true,
		FirstStepID:  &step1ID,
	}
	require.NoError(t, db.Create(&workflow).Error)

	require.NoError(t, db.Create(&models.WorkflowStep{
		ID:                step1ID,
		WorkflowID:        workflowID,
		StepOrder:         1,
		StepName:          "step1",
		ResponsibleRole:   models.RoleManager,
		ActionType:        models.ActionApprove,
		NextStepOnApprove: &step2ID,
	}).Error)
	require.NoError(t, db.Create(&models.WorkflowStep{
		ID:              step2ID,
		WorkflowID:      workflowID,
		StepOrder:       2,
		StepName:        "step2",
		ResponsibleRole: models.RoleHR,
		ActionType:      models.ActionApprove,
		IsTerminal:      true,
		TerminalStatus:  models.StatusApproved,
	}).Error)

	req := models.LeaveRequest{ID: uuid.New(), UserID: uuid.New(), LeaveType: models.LeaveTypeAnnual}
	require.NoError(t, db.Create(&req).Error)
	_, err = workflowSvc.InitializeWorkflowState(&req)
	require.NoError(t, err)

	newStep := models.WorkflowStep{
		WorkflowID:        workflowID,
		StepOrder:         2,
		StepName:          "inserted_step",
		ResponsibleRole:   models.RoleHR,
		ActionType:        models.ActionApprove,
		NextStepOnApprove: &step2ID,
	}
	require.NoError(t, workflowSvc.CreateWorkflowStep(&newStep))

	var activeWorkflow models.LeaveWorkflow
	require.NoError(t, db.Where("leave_type = ? AND is_active = ?", models.LeaveTypeAnnual, true).First(&activeWorkflow).Error)
	assert.NotEqual(t, workflowID, activeWorkflow.ID)

	var clonedStep2 models.WorkflowStep
	require.NoError(t, db.Where("workflow_id = ? AND step_name = ?", activeWorkflow.ID, "step2").First(&clonedStep2).Error)

	var inserted models.WorkflowStep
	require.NoError(t, db.Where("workflow_id = ? AND step_name = ?", activeWorkflow.ID, "inserted_step").First(&inserted).Error)
	require.NotNil(t, inserted.NextStepOnApprove)
	assert.Equal(t, clonedStep2.ID, *inserted.NextStepOnApprove)
	assert.NotEqual(t, step2ID, *inserted.NextStepOnApprove)
}
