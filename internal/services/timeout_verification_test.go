package services_test

import (
	"leave-management-system/internal/models"
	"leave-management-system/internal/services"
	"testing"
	"time"

	"github.com/glebarez/sqlite"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

func TestTimeoutLogic(t *testing.T) {
	// Setup DB
	db, err := gorm.Open(sqlite.Open("file::memory:?cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatalf("Failed to connect to database: %v", err)
	}
	db.AutoMigrate(&models.User{}, &models.LeaveRequest{}, &models.LeaveWorkflow{}, &models.WorkflowStep{}, &models.LeaveRequestWorkflowState{}, &models.LeaveBalance{}, &models.Chronology{})

	// Setup Service
	wfService := services.NewWorkflowService(db, nil) // no department svc needed for this test

	// 1. Create a Workflow with a 1-day Timeout -> AutoApprove
	workflowID := uuid.New()
	stepID := uuid.New()
	wf := models.LeaveWorkflow{
		ID:           workflowID,
		LeaveType:    models.LeaveTypeSick,
		WorkflowName: "Sick Leave Timeout Test",
		FirstStepID:  &stepID,
		IsActive:     true,
		Version:      1,
	}
	db.Create(&wf)

	step := models.WorkflowStep{
		ID:              stepID,
		WorkflowID:      workflowID,
		StepName:        "timeout_step",
		StepLabel:       "Timeout Step",
		ResponsibleRole: models.RoleManager,
		ActionType:      models.ActionApprove,
		TimeoutDays:     1,
		TimeoutAction:   models.TimeoutAutoApprove,
		IsTerminal:      true,
		TerminalStatus:  models.StatusApproved,
		CreatedAt:       time.Now(),
	}
	db.Create(&step)

	// 2. Create a Request
	reqID := uuid.New()
	userID := uuid.New()
	req := models.LeaveRequest{
		ID:        reqID,
		UserID:    userID,
		LeaveType: models.LeaveTypeSick,
		Status:    models.StatusPending,
	}
	db.Create(&req)

	// 3. Create a Workflow State that STARTED 2 DAYS AGO (timed out)
	twoDaysAgo := time.Now().Add(-48 * time.Hour)
	state := models.LeaveRequestWorkflowState{
		ID:             uuid.New(),
		LeaveRequestID: reqID,
		WorkflowID:     workflowID,
		CurrentStepID:  &stepID,
		CurrentStep:    &step,
		StepStartedAt:  twoDaysAgo,
		ActionTaken:    models.StepActionPending,
		IsComplete:     false,
	}
	db.Create(&state)

	// Update request to point to state
	req.WorkflowStateID = &state.ID
	db.Save(&req)

	// 4. Run ProcessTimeouts
	processedIDs, err := wfService.ProcessTimeouts()
	if err != nil {
		t.Fatalf("ProcessTimeouts failed: %v", err)
	}

	// 5. Verify the request ID is returned
	if len(processedIDs) != 1 || processedIDs[0] != reqID {
		t.Errorf("Expected request %v to be processed, got %v", reqID, processedIDs)
	}

	// 6. Verify the state is updated (Auto-Approved)
	var updatedState models.LeaveRequestWorkflowState
	db.First(&updatedState, "id = ?", state.ID)

	if !updatedState.IsComplete {
		t.Error("Workflow should be complete after auto-approve")
	}
	if updatedState.FinalStatus != models.StatusApproved {
		t.Errorf("Expected final status Approved, got %s", updatedState.FinalStatus)
	}
	if updatedState.ActionTaken != models.StepActionApproved {
		t.Errorf("Expected action Approved, got %s", updatedState.ActionTaken)
	}
}
