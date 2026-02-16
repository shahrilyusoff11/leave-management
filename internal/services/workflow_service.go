package services

import (
	"errors"
	"fmt"
	"leave-management-system/internal/models"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// WorkflowService handles leave workflow operations
type WorkflowService struct {
	db            *gorm.DB
	departmentSvc *DepartmentService
}

func NewWorkflowService(db *gorm.DB, departmentSvc *DepartmentService) *WorkflowService {
	return &WorkflowService{db: db, departmentSvc: departmentSvc}
}

// GetWorkflowForLeaveType returns the active workflow configuration for a leave type (used at runtime)
func (s *WorkflowService) GetWorkflowForLeaveType(leaveType models.LeaveType) (*models.LeaveWorkflow, error) {
	var workflow models.LeaveWorkflow
	err := s.db.Preload("Steps", func(db *gorm.DB) *gorm.DB {
		return db.Order("step_order ASC")
	}).Where("leave_type = ? AND is_active = ?", leaveType, true).First(&workflow).Error
	if err != nil {
		return nil, err
	}
	return &workflow, nil
}

// GetWorkflowByLeaveType returns the workflow for a leave type regardless of active status (used by admin)
func (s *WorkflowService) GetWorkflowByLeaveType(leaveType models.LeaveType) (*models.LeaveWorkflow, error) {
	var workflow models.LeaveWorkflow
	err := s.db.Preload("Steps", func(db *gorm.DB) *gorm.DB {
		return db.Order("step_order ASC")
	}).Where("leave_type = ?", leaveType).First(&workflow).Error
	if err != nil {
		return nil, err
	}
	return &workflow, nil
}

// GetAllWorkflows returns all workflow configurations
func (s *WorkflowService) GetAllWorkflows() ([]models.LeaveWorkflow, error) {
	var workflows []models.LeaveWorkflow
	err := s.db.Preload("Steps", func(db *gorm.DB) *gorm.DB {
		return db.Order("step_order ASC")
	}).Find(&workflows).Error
	return workflows, err
}

// GetWorkflowStep returns a specific workflow step
func (s *WorkflowService) GetWorkflowStep(stepID uuid.UUID) (*models.WorkflowStep, error) {
	var step models.WorkflowStep
	err := s.db.First(&step, "id = ?", stepID).Error
	if err != nil {
		return nil, err
	}
	return &step, nil
}

// InitializeWorkflowState creates initial workflow state for a leave request
func (s *WorkflowService) InitializeWorkflowState(leaveRequest *models.LeaveRequest) (*models.LeaveRequestWorkflowState, error) {
	workflow, err := s.GetWorkflowForLeaveType(leaveRequest.LeaveType)
	if err != nil {
		return nil, fmt.Errorf("no active workflow configuration found for leave type %s: %w", leaveRequest.LeaveType, err)
	}

	if workflow.FirstStepID == nil {
		return nil, errors.New("workflow has no first step configured")
	}

	state := &models.LeaveRequestWorkflowState{
		ID:             uuid.New(),
		LeaveRequestID: leaveRequest.ID,
		WorkflowID:     workflow.ID,
		CurrentStepID:  workflow.FirstStepID,
		StepStartedAt:  time.Now(),
		ActionTaken:    models.StepActionPending,
		StepHistory:    models.JSONMap{"steps": []interface{}{}},
		IsComplete:     false,
		CreatedAt:      time.Now(),
		UpdatedAt:      time.Now(),
	}

	if err := s.db.Create(state).Error; err != nil {
		return nil, err
	}

	// Update leave request with workflow state ID
	leaveRequest.WorkflowStateID = &state.ID

	return state, nil
}

// GetWorkflowState returns the current workflow state for a leave request
func (s *WorkflowService) GetWorkflowState(leaveRequestID uuid.UUID) (*models.LeaveRequestWorkflowState, error) {
	var state models.LeaveRequestWorkflowState
	err := s.db.Preload("CurrentStep").Where("leave_request_id = ?", leaveRequestID).First(&state).Error
	if err != nil {
		return nil, err
	}
	return &state, nil
}

// ProcessAction handles a workflow action (approve, reject, verify, etc.)
func (s *WorkflowService) ProcessAction(
	leaveRequestID uuid.UUID,
	action models.WorkflowStepAction,
	actorID uuid.UUID,
	comment string,
) (*models.LeaveRequestWorkflowState, error) {
	var state *models.LeaveRequestWorkflowState
	err := s.db.Transaction(func(tx *gorm.DB) error {
		var err error
		state, err = s.ProcessActionWithTx(tx, leaveRequestID, action, actorID, comment)
		return err
	})
	return state, err
}

// CancelWorkflow terminates a workflow process
func (s *WorkflowService) CancelWorkflow(leaveRequestID uuid.UUID, actorID uuid.UUID, reason string) error {
	return s.db.Transaction(func(tx *gorm.DB) error {
		return s.CancelWorkflowWithTx(tx, leaveRequestID, actorID, reason)
	})
}

// CancelWorkflowWithTx terminates a workflow process within an existing transaction
func (s *WorkflowService) CancelWorkflowWithTx(tx *gorm.DB, leaveRequestID uuid.UUID, actorID uuid.UUID, reason string) error {
	// Get current state
	var state models.LeaveRequestWorkflowState
	if err := tx.Preload("CurrentStep").Where("leave_request_id = ?", leaveRequestID).First(&state).Error; err != nil {
		return fmt.Errorf("workflow state not found: %w", err)
	}

	if state.IsComplete {
		return errors.New("workflow is already complete")
	}

	// Record action
	state.ActionTaken = models.StepActionCancelled
	state.ActionBy = &actorID
	state.ActionComment = reason

	// Add to step history if current step exists
	if state.CurrentStep != nil {
		historyEntry := map[string]interface{}{
			"step_id":   state.CurrentStep.ID.String(),
			"step_name": state.CurrentStep.StepName,
			"action":    "cancelled",
			"actor_id":  actorID.String(),
			"comment":   reason,
			"completed": time.Now().Format(time.RFC3339),
		}
		s.addToStepHistory(&state, historyEntry)
	}

	// Update state to complete
	state.IsComplete = true
	now := time.Now()
	state.CompletedAt = &now
	state.FinalStatus = models.StatusCancelled
	state.CurrentStepID = nil
	state.UpdatedAt = time.Now()

	return tx.Save(&state).Error
}

func (s *WorkflowService) ProcessActionWithTx(
	tx *gorm.DB,
	leaveRequestID uuid.UUID,
	action models.WorkflowStepAction,
	actorID uuid.UUID,
	comment string,
) (*models.LeaveRequestWorkflowState, error) {
	// Get current state
	var state models.LeaveRequestWorkflowState
	if err := tx.Preload("CurrentStep").Where("leave_request_id = ?", leaveRequestID).First(&state).Error; err != nil {
		return nil, fmt.Errorf("workflow state not found: %w", err)
	}

	if state.IsComplete {
		return nil, errors.New("workflow is already complete")
	}

	if state.CurrentStepID == nil || state.CurrentStep == nil {
		return nil, errors.New("no current step in workflow")
	}

	currentStep := state.CurrentStep

	// Record action
	state.ActionTaken = action
	state.ActionBy = &actorID
	state.ActionComment = comment

	// Add to step history
	historyEntry := map[string]interface{}{
		"step_id":   currentStep.ID.String(),
		"step_name": currentStep.StepName,
		"action":    string(action),
		"actor_id":  actorID.String(),
		"comment":   comment,
		"completed": time.Now().Format(time.RFC3339),
	}
	s.addToStepHistory(&state, historyEntry)

	// Determine next step based on action
	var nextStepID *uuid.UUID
	var isTerminal bool
	var terminalStatus models.LeaveStatus

	switch action {
	case models.StepActionApproved, models.StepActionVerified:
		nextStepID = currentStep.NextStepOnApprove
		if currentStep.IsTerminal {
			isTerminal = true
			terminalStatus = currentStep.TerminalStatus
		}
	case models.StepActionRejected, models.StepActionNotVerified:
		nextStepID = currentStep.NextStepOnReject
		if currentStep.NextStepOnReject == nil {
			isTerminal = true
			terminalStatus = models.StatusRejected
		}
	case models.StepActionCategorizedAL, models.StepActionCategorizedUL:
		// For categorization steps, move to next step
		nextStepID = currentStep.NextStepOnApprove
		isTerminal = currentStep.IsTerminal
		if isTerminal {
			terminalStatus = models.StatusApproved
		}
	case models.StepActionRequestedDocs:
		// Stay at current step, waiting for document resubmission
		state.ActionTaken = models.StepActionPending
		state.StepStartedAt = time.Now()
	case models.StepActionEscalated:
		// Move to fallback step if configured
		if currentStep.FallbackStepID != nil {
			nextStepID = currentStep.FallbackStepID
		} else {
			isTerminal = true
			terminalStatus = models.StatusEscalated
		}
	default:
		return nil, fmt.Errorf("unknown action: %s", action)
	}

	// Update state
	if isTerminal || (nextStepID == nil && currentStep.IsTerminal) {
		state.IsComplete = true
		now := time.Now()
		state.CompletedAt = &now
		state.FinalStatus = terminalStatus
		state.CurrentStepID = nil
	} else if nextStepID != nil {
		state.PreviousStepID = state.CurrentStepID
		state.CurrentStepID = nextStepID
		state.StepStartedAt = time.Now()
		state.ActionTaken = models.StepActionPending
	}

	state.UpdatedAt = time.Now()

	// Nil out CurrentStep to ensure GORM updates the ID column and doesn't try to sync with the old loaded struct
	state.CurrentStep = nil

	if err := tx.Save(&state).Error; err != nil {
		return nil, err
	}

	// Reload with current step
	if state.CurrentStepID != nil {
		tx.Preload("CurrentStep").First(&state, "id = ?", state.ID)
	}

	return &state, nil
}

// ProcessTimeouts checks for overdue workflow steps and applies timeout actions
func (s *WorkflowService) ProcessTimeouts() ([]uuid.UUID, error) {
	var escalatedRequests []uuid.UUID

	// Find all pending workflow states
	var states []models.LeaveRequestWorkflowState
	err := s.db.Preload("CurrentStep").
		Where("is_complete = ? AND current_step_id IS NOT NULL", false).
		Find(&states).Error
	if err != nil {
		return nil, err
	}

	for _, state := range states {
		if state.CurrentStep == nil {
			continue
		}

		step := state.CurrentStep
		timeoutDuration := time.Duration(step.TimeoutDays) * 24 * time.Hour
		deadline := state.StepStartedAt.Add(timeoutDuration)

		if time.Now().After(deadline) {
			// Step has timed out - apply timeout action
			requestID, err := s.applyTimeoutAction(&state, step)
			if err != nil {
				continue // Log error and continue with other states
			}
			if requestID != uuid.Nil {
				escalatedRequests = append(escalatedRequests, requestID)
			}
		}
	}

	return escalatedRequests, nil
}

func (s *WorkflowService) applyTimeoutAction(state *models.LeaveRequestWorkflowState, step *models.WorkflowStep) (uuid.UUID, error) {
	switch step.TimeoutAction {
	case models.TimeoutEscalate:
		_, err := s.ProcessAction(state.LeaveRequestID, models.StepActionEscalated, uuid.Nil, "Auto-escalated due to timeout")
		return state.LeaveRequestID, err

	case models.TimeoutAutoApprove:
		_, err := s.ProcessAction(state.LeaveRequestID, models.StepActionApproved, uuid.Nil, "Auto-approved due to timeout")
		return state.LeaveRequestID, err

	case models.TimeoutFallback:
		if step.FallbackStepID != nil {
			state.PreviousStepID = state.CurrentStepID
			state.CurrentStepID = step.FallbackStepID
			state.StepStartedAt = time.Now()
			state.ActionTaken = models.StepActionTimeoutApplied
			state.UpdatedAt = time.Now()
			return state.LeaveRequestID, s.db.Save(state).Error
		}
		return uuid.Nil, nil

	case models.TimeoutConvert:
		if step.ConvertToType != nil {
			return state.LeaveRequestID, s.convertLeaveType(state.LeaveRequestID, *step.ConvertToType)
		}
		return uuid.Nil, nil

	default:
		return uuid.Nil, nil
	}
}

// ConvertLeaveType changes the leave type of a request (e.g., AL→EL)
func (s *WorkflowService) ConvertLeaveType(leaveRequestID uuid.UUID, newType models.LeaveType, actorID uuid.UUID, reason string) error {
	return s.db.Transaction(func(tx *gorm.DB) error {
		var request models.LeaveRequest
		if err := tx.First(&request, "id = ?", leaveRequestID).Error; err != nil {
			return err
		}

		// Store original type
		originalType := request.LeaveType
		request.ConvertedFrom = &originalType
		request.LeaveType = newType
		request.UpdatedAt = time.Now()

		if err := tx.Save(&request).Error; err != nil {
			return err
		}

		// Create chronology entry
		chronology := models.Chronology{
			ID:             uuid.New(),
			LeaveRequestID: leaveRequestID,
			Action:         "converted",
			ActorID:        actorID,
			Comment:        reason,
			Metadata: models.JSONMap{
				"from_type": string(originalType),
				"to_type":   string(newType),
			},
			CreatedAt: time.Now(),
		}

		if err := tx.Create(&chronology).Error; err != nil {
			return err
		}

		// Re-initialize workflow for new leave type
		newWorkflow, err := s.GetWorkflowForLeaveType(newType)
		if err != nil {
			return nil // No workflow for new type, continue without
		}

		// Update workflow state
		var state models.LeaveRequestWorkflowState
		if err := tx.Where("leave_request_id = ?", leaveRequestID).First(&state).Error; err == nil {
			state.WorkflowID = newWorkflow.ID
			state.CurrentStepID = newWorkflow.FirstStepID
			state.StepStartedAt = time.Now()
			state.ActionTaken = models.StepActionPending
			state.UpdatedAt = time.Now()
			return tx.Save(&state).Error
		}

		return nil
	})
}

func (s *WorkflowService) convertLeaveType(leaveRequestID uuid.UUID, newType models.LeaveType) error {
	return s.ConvertLeaveType(leaveRequestID, newType, uuid.Nil, "Auto-converted due to timeout")
}

// GetResponsibleUsers returns users who can act on the current step.
// For HOD steps, it resolves the department's HOD (or acting HOD via delegation).
func (s *WorkflowService) GetResponsibleUsers(state *models.LeaveRequestWorkflowState) ([]models.User, error) {
	if state.CurrentStep == nil {
		return nil, errors.New("no current step")
	}

	// If the responsible role is HOD, route to the applicant's department HOD
	if state.CurrentStep.ResponsibleRole == models.RoleHOD && s.departmentSvc != nil {
		// Get the leave request to find the applicant
		var leaveRequest models.LeaveRequest
		if err := s.db.Preload("User").First(&leaveRequest, "id = ?", state.LeaveRequestID).Error; err != nil {
			return nil, fmt.Errorf("failed to load leave request: %w", err)
		}

		if leaveRequest.User.DepartmentID != nil {
			approver, err := s.departmentSvc.ResolveApproverForDepartment(*leaveRequest.User.DepartmentID)
			if err == nil && approver != nil {
				return []models.User{*approver}, nil
			}
			// If resolution fails, fall through to role-based lookup
		}
	}

	// Default: role-based lookup
	var users []models.User
	err := s.db.Where("role = ? AND is_active = ?", state.CurrentStep.ResponsibleRole, true).Find(&users).Error
	return users, err
}

// addToStepHistory appends an entry to the step history
func (s *WorkflowService) addToStepHistory(state *models.LeaveRequestWorkflowState, entry map[string]interface{}) {
	if state.StepHistory == nil {
		state.StepHistory = models.JSONMap{"steps": []interface{}{}}
	}

	steps, ok := state.StepHistory["steps"].([]interface{})
	if !ok {
		steps = []interface{}{}
	}
	steps = append(steps, entry)
	state.StepHistory["steps"] = steps
}

// UpdateWorkflow updates a workflow configuration
func (s *WorkflowService) UpdateWorkflow(workflow *models.LeaveWorkflow) error {
	workflow.UpdatedAt = time.Now()
	return s.db.Save(workflow).Error
}

// CreateWorkflowStep creates a new step in a workflow
func (s *WorkflowService) CreateWorkflowStep(step *models.WorkflowStep) error {
	step.ID = uuid.New()
	step.CreatedAt = time.Now()
	step.UpdatedAt = time.Now()
	return s.db.Create(step).Error
}

// UpdateWorkflowStep updates a workflow step
func (s *WorkflowService) UpdateWorkflowStep(step *models.WorkflowStep) error {
	step.UpdatedAt = time.Now()
	return s.db.Save(step).Error
}

// DeleteWorkflowStep removes a step from a workflow
func (s *WorkflowService) DeleteWorkflowStep(stepID uuid.UUID) error {
	return s.db.Delete(&models.WorkflowStep{}, "id = ?", stepID).Error
}

// ReorderWorkflowSteps updates the order of steps in a workflow
func (s *WorkflowService) ReorderWorkflowSteps(workflowID uuid.UUID, stepOrders map[uuid.UUID]int) error {
	return s.db.Transaction(func(tx *gorm.DB) error {
		for stepID, order := range stepOrders {
			if err := tx.Model(&models.WorkflowStep{}).
				Where("id = ? AND workflow_id = ?", stepID, workflowID).
				Update("step_order", order).Error; err != nil {
				return err
			}
		}
		return nil
	})
}

// EvaluateConditions checks if step conditions are met
func (s *WorkflowService) EvaluateConditions(step *models.WorkflowStep, request *models.LeaveRequest) (bool, string) {
	if len(step.Conditions) == 0 {
		return true, ""
	}

	// Check advance notice condition for maternity/paternity
	if minDays, ok := step.Conditions["min_advance_days"].(float64); ok {
		daysInAdvance := request.StartDate.Sub(request.CreatedAt).Hours() / 24
		if daysInAdvance < minDays {
			return false, fmt.Sprintf("Application submitted %.0f days in advance, requires %.0f days", daysInAdvance, minDays)
		}
	}

	// Check document requirement
	if requiresDoc, ok := step.Conditions["requires_document"].(bool); ok && requiresDoc {
		if request.AttachmentURL == "" {
			docType := "supporting document"
			if dt, ok := step.Conditions["document_type"].(string); ok {
				docType = dt
			}
			return false, fmt.Sprintf("Missing required %s", docType)
		}
	}

	return true, ""
}

// SeedDefaultWorkflows creates default workflow configurations for all leave types
func (s *WorkflowService) SeedDefaultWorkflows() error {
	var count int64
	s.db.Model(&models.LeaveWorkflow{}).Count(&count)
	if count > 0 {
		return nil // Already seeded
	}

	// Annual Leave Workflow
	if err := s.seedAnnualLeaveWorkflow(); err != nil {
		return err
	}

	// Emergency Leave Workflow (same as Annual)
	if err := s.seedEmergencyLeaveWorkflow(); err != nil {
		return err
	}

	// Sick Leave Workflow
	if err := s.seedSickLeaveWorkflow(); err != nil {
		return err
	}

	// Hospitalization Leave (same as Sick)
	if err := s.seedHospitalizationLeaveWorkflow(); err != nil {
		return err
	}

	// Maternity Leave Workflow
	if err := s.seedMaternityLeaveWorkflow(); err != nil {
		return err
	}

	// Paternity Leave (same as Maternity)
	if err := s.seedPaternityLeaveWorkflow(); err != nil {
		return err
	}

	// Unrecorded Leave Workflow
	if err := s.seedUnrecordedLeaveWorkflow(); err != nil {
		return err
	}

	// Unpaid Leave Workflow
	if err := s.seedUnpaidLeaveWorkflow(); err != nil {
		return err
	}

	return nil
}

func (s *WorkflowService) seedAnnualLeaveWorkflow() error {
	workflowID := uuid.New()
	hodStepID := uuid.New()
	approvedStepID := uuid.New()

	workflow := models.LeaveWorkflow{
		ID:           workflowID,
		LeaveType:    models.LeaveTypeAnnual,
		WorkflowName: "Annual Leave Approval",
		Description:  "HOD approval with 7-day timeout for conversion decision",
		FirstStepID:  &hodStepID,
		IsActive:     true,
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	}

	steps := []models.WorkflowStep{
		{
			ID:              hodStepID,
			WorkflowID:      workflowID,
			StepOrder:       1,
			StepName:        "hod_approval",
			StepLabel:       "HOD Approval",
			ResponsibleRole: models.RoleHOD,
			ActionType:      models.ActionApprove,
			TimeoutDays:     7,
			// Simplified: No auto-actions, just wait for approval
			NextStepOnApprove: &approvedStepID,
			NotifyRoles:       models.JSONArray{"manager"},
			CreatedAt:         time.Now(),
			UpdatedAt:         time.Now(),
		},
		{
			ID:              approvedStepID,
			WorkflowID:      workflowID,
			StepOrder:       2,
			StepName:        "approved",
			StepLabel:       "Approved",
			ResponsibleRole: models.RoleHOD,
			ActionType:      models.ActionApprove,
			IsTerminal:      true,
			TerminalStatus:  models.StatusApproved,
			NotifyRoles:     models.JSONArray{"manager"},
			CreatedAt:       time.Now(),
			UpdatedAt:       time.Now(),
		},
	}

	if err := s.db.Create(&workflow).Error; err != nil {
		return err
	}
	for _, step := range steps {
		if err := s.db.Create(&step).Error; err != nil {
			return err
		}
	}
	return nil
}

func (s *WorkflowService) seedEmergencyLeaveWorkflow() error {
	workflowID := uuid.New()
	hodStepID := uuid.New()
	hrStepID := uuid.New()
	approvedStepID := uuid.New()
	unpaidStepID := uuid.New()

	workflow := models.LeaveWorkflow{
		ID:           workflowID,
		LeaveType:    models.LeaveTypeEmergency,
		WorkflowName: "Emergency Leave Approval",
		Description:  "HOD approval, if rejected by HR converts to Unpaid Leave",
		FirstStepID:  &hodStepID,
		IsActive:     true,
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	}

	steps := []models.WorkflowStep{
		{
			ID:              hodStepID,
			WorkflowID:      workflowID,
			StepOrder:       1,
			StepName:        "hod_approval",
			StepLabel:       "HOD Approval",
			ResponsibleRole: models.RoleHOD,
			ActionType:      models.ActionApprove,
			TimeoutDays:     7,
			// Simplified: No escalation
			NextStepOnApprove: &hrStepID,
			NotifyRoles:       models.JSONArray{"manager"},
			CreatedAt:         time.Now(),
			UpdatedAt:         time.Now(),
		},
		{
			ID:                hrStepID,
			WorkflowID:        workflowID,
			StepOrder:         2,
			StepName:          "hr_review",
			StepLabel:         "HR Review",
			ResponsibleRole:   models.RoleHR,
			ActionType:        models.ActionApprove,
			TimeoutDays:       3,
			NextStepOnApprove: &approvedStepID,
			NextStepOnReject:  &unpaidStepID,
			NotifyRoles:       models.JSONArray{"manager"},
			CreatedAt:         time.Now(),
			UpdatedAt:         time.Now(),
		},
		{
			ID:              approvedStepID,
			WorkflowID:      workflowID,
			StepOrder:       3,
			StepName:        "approved",
			StepLabel:       "Approved",
			ResponsibleRole: models.RoleHR,
			ActionType:      models.ActionApprove,
			IsTerminal:      true,
			TerminalStatus:  models.StatusApproved,
			NotifyRoles:     models.JSONArray{"manager"},
			CreatedAt:       time.Now(),
			UpdatedAt:       time.Now(),
		},
		{
			ID:              unpaidStepID,
			WorkflowID:      workflowID,
			StepOrder:       4,
			StepName:        "convert_unpaid",
			StepLabel:       "Converted to Unpaid Leave",
			ResponsibleRole: models.RoleHR,
			ActionType:      models.ActionCategorize,
			IsTerminal:      true,
			TerminalStatus:  models.StatusApproved,
			NotifyRoles:     models.JSONArray{"manager"},
			CreatedAt:       time.Now(),
			UpdatedAt:       time.Now(),
		},
	}

	if err := s.db.Create(&workflow).Error; err != nil {
		return err
	}
	for _, step := range steps {
		if err := s.db.Create(&step).Error; err != nil {
			return err
		}
	}
	return nil
}

func (s *WorkflowService) seedSickLeaveWorkflow() error {
	workflowID := uuid.New()
	hodVerifyID := uuid.New()
	hrClarifyID := uuid.New()
	hrReviewID := uuid.New()
	requestDocsID := uuid.New()
	approvedStepID := uuid.New()

	workflow := models.LeaveWorkflow{
		ID:           workflowID,
		LeaveType:    models.LeaveTypeSick,
		WorkflowName: "Sick Leave Approval",
		Description:  "HOD verification, HR MC check",
		FirstStepID:  &hodVerifyID,
		IsActive:     true,
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	}

	steps := []models.WorkflowStep{
		{
			ID:                hodVerifyID,
			WorkflowID:        workflowID,
			StepOrder:         1,
			StepName:          "hod_verify",
			StepLabel:         "HOD Verification",
			ResponsibleRole:   models.RoleHOD,
			ActionType:        models.ActionVerify,
			TimeoutDays:       3,
			TimeoutAction:     models.TimeoutFallback,
			FallbackStepID:    &hrClarifyID,
			NextStepOnApprove: &hrReviewID,
			NextStepOnReject:  &hrClarifyID,
			CreatedAt:         time.Now(),
			UpdatedAt:         time.Now(),
		},
		{
			ID:                hrClarifyID,
			WorkflowID:        workflowID,
			StepOrder:         2,
			StepName:          "hr_clarify",
			StepLabel:         "HR Clarification",
			ResponsibleRole:   models.RoleHR,
			ActionType:        models.ActionReview,
			TimeoutDays:       3,
			NextStepOnApprove: &hrReviewID,
			CreatedAt:         time.Now(),
			UpdatedAt:         time.Now(),
		},
		{
			ID:                hrReviewID,
			WorkflowID:        workflowID,
			StepOrder:         3,
			StepName:          "hr_review",
			StepLabel:         "HR MC Validation",
			ResponsibleRole:   models.RoleHR,
			ActionType:        models.ActionReview,
			TimeoutDays:       3,
			RequiresDocument:  true,
			DocumentType:      "medical_certificate",
			NextStepOnApprove: &approvedStepID,
			NextStepOnReject:  &requestDocsID,
			Conditions:        models.JSONMap{"requires_document": true, "document_type": "MC"},
			CreatedAt:         time.Now(),
			UpdatedAt:         time.Now(),
		},
		{
			ID:                requestDocsID,
			WorkflowID:        workflowID,
			StepOrder:         4,
			StepName:          "request_docs",
			StepLabel:         "Request Additional Documents",
			ResponsibleRole:   models.RoleHR,
			ActionType:        models.ActionReview,
			TimeoutDays:       7,
			NextStepOnApprove: &hrReviewID,
			CreatedAt:         time.Now(),
			UpdatedAt:         time.Now(),
		},
		{
			ID:              approvedStepID,
			WorkflowID:      workflowID,
			StepOrder:       5,
			StepName:        "approved",
			StepLabel:       "Approved",
			ResponsibleRole: models.RoleHR,
			ActionType:      models.ActionApprove,
			IsTerminal:      true,
			TerminalStatus:  models.StatusApproved,
			NotifyRoles:     models.JSONArray{"manager"},
			CreatedAt:       time.Now(),
			UpdatedAt:       time.Now(),
		},
	}

	if err := s.db.Create(&workflow).Error; err != nil {
		return err
	}
	for _, step := range steps {
		if err := s.db.Create(&step).Error; err != nil {
			return err
		}
	}
	return nil
}

func (s *WorkflowService) seedHospitalizationLeaveWorkflow() error {
	// Same structure as Sick Leave
	workflowID := uuid.New()
	hodVerifyID := uuid.New()
	hrClarifyID := uuid.New()
	hrReviewID := uuid.New()
	requestDocsID := uuid.New()
	approvedStepID := uuid.New()

	workflow := models.LeaveWorkflow{
		ID:           workflowID,
		LeaveType:    models.LeaveTypeHospitalization,
		WorkflowName: "Hospitalization Leave Approval",
		Description:  "HOD verification, HR document check",
		FirstStepID:  &hodVerifyID,
		IsActive:     true,
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	}

	steps := []models.WorkflowStep{
		{
			ID:                hodVerifyID,
			WorkflowID:        workflowID,
			StepOrder:         1,
			StepName:          "hod_verify",
			StepLabel:         "HOD Verification",
			ResponsibleRole:   models.RoleHOD,
			ActionType:        models.ActionVerify,
			TimeoutDays:       3,
			TimeoutAction:     models.TimeoutFallback,
			FallbackStepID:    &hrClarifyID,
			NextStepOnApprove: &hrReviewID,
			NextStepOnReject:  &hrClarifyID,
			CreatedAt:         time.Now(),
			UpdatedAt:         time.Now(),
		},
		{
			ID:                hrClarifyID,
			WorkflowID:        workflowID,
			StepOrder:         2,
			StepName:          "hr_clarify",
			StepLabel:         "HR Clarification",
			ResponsibleRole:   models.RoleHR,
			ActionType:        models.ActionReview,
			TimeoutDays:       3,
			NextStepOnApprove: &hrReviewID,
			CreatedAt:         time.Now(),
			UpdatedAt:         time.Now(),
		},
		{
			ID:                hrReviewID,
			WorkflowID:        workflowID,
			StepOrder:         3,
			StepName:          "hr_review",
			StepLabel:         "HR Document Validation",
			ResponsibleRole:   models.RoleHR,
			ActionType:        models.ActionReview,
			TimeoutDays:       3,
			RequiresDocument:  true,
			DocumentType:      "hospital_docs",
			NextStepOnApprove: &approvedStepID,
			NextStepOnReject:  &requestDocsID,
			CreatedAt:         time.Now(),
			UpdatedAt:         time.Now(),
		},
		{
			ID:                requestDocsID,
			WorkflowID:        workflowID,
			StepOrder:         4,
			StepName:          "request_docs",
			StepLabel:         "Request Additional Documents",
			ResponsibleRole:   models.RoleHR,
			ActionType:        models.ActionReview,
			TimeoutDays:       7,
			NextStepOnApprove: &hrReviewID,
			CreatedAt:         time.Now(),
			UpdatedAt:         time.Now(),
		},
		{
			ID:              approvedStepID,
			WorkflowID:      workflowID,
			StepOrder:       5,
			StepName:        "approved",
			StepLabel:       "Approved",
			ResponsibleRole: models.RoleHR,
			ActionType:      models.ActionApprove,
			IsTerminal:      true,
			TerminalStatus:  models.StatusApproved,
			NotifyRoles:     models.JSONArray{"manager"},
			CreatedAt:       time.Now(),
			UpdatedAt:       time.Now(),
		},
	}

	if err := s.db.Create(&workflow).Error; err != nil {
		return err
	}
	for _, step := range steps {
		if err := s.db.Create(&step).Error; err != nil {
			return err
		}
	}
	return nil
}

func (s *WorkflowService) seedMaternityLeaveWorkflow() error {
	workflowID := uuid.New()
	checkTimingID := uuid.New()
	justificationID := uuid.New()
	hrReviewID := uuid.New()
	hodApprovalID := uuid.New()
	requestDocsID := uuid.New()
	approvedStepID := uuid.New()

	workflow := models.LeaveWorkflow{
		ID:           workflowID,
		LeaveType:    models.LeaveTypeMaternity,
		WorkflowName: "Maternity Leave Approval",
		Description:  "2-month advance submission check, HR review, HOD approval",
		FirstStepID:  &checkTimingID,
		IsActive:     true,
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	}

	steps := []models.WorkflowStep{
		{
			ID:                checkTimingID,
			WorkflowID:        workflowID,
			StepOrder:         1,
			StepName:          "check_timing",
			StepLabel:         "Submission Timing Check",
			ResponsibleRole:   models.RoleHR,
			ActionType:        models.ActionReview,
			TimeoutDays:       1,
			TimeoutAction:     models.TimeoutAutoApprove,
			NextStepOnApprove: &hrReviewID,
			NextStepOnReject:  &justificationID,
			Conditions:        models.JSONMap{"min_advance_days": float64(60)},
			NotifyRoles:       models.JSONArray{"manager", "hod"},
			CreatedAt:         time.Now(),
			UpdatedAt:         time.Now(),
		},
		{
			ID:                justificationID,
			WorkflowID:        workflowID,
			StepOrder:         2,
			StepName:          "justification",
			StepLabel:         "Employee Justification Required",
			ResponsibleRole:   models.RoleStaff,
			ActionType:        models.ActionSubmit,
			TimeoutDays:       7,
			NextStepOnApprove: &hrReviewID,
			CreatedAt:         time.Now(),
			UpdatedAt:         time.Now(),
		},
		{
			ID:                hrReviewID,
			WorkflowID:        workflowID,
			StepOrder:         3,
			StepName:          "hr_review",
			StepLabel:         "HR Document Review",
			ResponsibleRole:   models.RoleHR,
			ActionType:        models.ActionReview,
			TimeoutDays:       5,
			RequiresDocument:  true,
			DocumentType:      "maternity_docs",
			NextStepOnApprove: &hodApprovalID,
			NextStepOnReject:  &requestDocsID,
			CreatedAt:         time.Now(),
			UpdatedAt:         time.Now(),
		},
		{
			ID:                requestDocsID,
			WorkflowID:        workflowID,
			StepOrder:         4,
			StepName:          "request_docs",
			StepLabel:         "Request Document Update",
			ResponsibleRole:   models.RoleHR,
			ActionType:        models.ActionReview,
			TimeoutDays:       7,
			NextStepOnApprove: &hrReviewID,
			CreatedAt:         time.Now(),
			UpdatedAt:         time.Now(),
		},
		{
			ID:                hodApprovalID,
			WorkflowID:        workflowID,
			StepOrder:         5,
			StepName:          "hod_approval",
			StepLabel:         "HOD Approval",
			ResponsibleRole:   models.RoleHOD,
			ActionType:        models.ActionApprove,
			TimeoutDays:       5,
			TimeoutAction:     models.TimeoutEscalate,
			NextStepOnApprove: &approvedStepID,
			CreatedAt:         time.Now(),
			UpdatedAt:         time.Now(),
		},
		{
			ID:              approvedStepID,
			WorkflowID:      workflowID,
			StepOrder:       6,
			StepName:        "approved",
			StepLabel:       "Approved",
			ResponsibleRole: models.RoleHR,
			ActionType:      models.ActionApprove,
			IsTerminal:      true,
			TerminalStatus:  models.StatusApproved,
			NotifyRoles:     models.JSONArray{"manager"},
			CreatedAt:       time.Now(),
			UpdatedAt:       time.Now(),
		},
	}

	if err := s.db.Create(&workflow).Error; err != nil {
		return err
	}
	for _, step := range steps {
		if err := s.db.Create(&step).Error; err != nil {
			return err
		}
	}
	return nil
}

func (s *WorkflowService) seedPaternityLeaveWorkflow() error {
	// Same as Maternity workflow
	workflowID := uuid.New()
	checkTimingID := uuid.New()
	justificationID := uuid.New()
	hrReviewID := uuid.New()
	hodApprovalID := uuid.New()
	requestDocsID := uuid.New()
	approvedStepID := uuid.New()

	workflow := models.LeaveWorkflow{
		ID:           workflowID,
		LeaveType:    models.LeaveTypePaternity,
		WorkflowName: "Paternity Leave Approval",
		Description:  "2-month advance submission check, HR review, HOD approval",
		FirstStepID:  &checkTimingID,
		IsActive:     true,
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	}

	steps := []models.WorkflowStep{
		{
			ID:                checkTimingID,
			WorkflowID:        workflowID,
			StepOrder:         1,
			StepName:          "check_timing",
			StepLabel:         "Submission Timing Check",
			ResponsibleRole:   models.RoleHR,
			ActionType:        models.ActionReview,
			TimeoutDays:       1,
			TimeoutAction:     models.TimeoutAutoApprove,
			NextStepOnApprove: &hrReviewID,
			NextStepOnReject:  &justificationID,
			Conditions:        models.JSONMap{"min_advance_days": float64(60)},
			NotifyRoles:       models.JSONArray{"manager", "hod"},
			CreatedAt:         time.Now(),
			UpdatedAt:         time.Now(),
		},
		{
			ID:                justificationID,
			WorkflowID:        workflowID,
			StepOrder:         2,
			StepName:          "justification",
			StepLabel:         "Employee Justification Required",
			ResponsibleRole:   models.RoleStaff,
			ActionType:        models.ActionSubmit,
			TimeoutDays:       7,
			NextStepOnApprove: &hrReviewID,
			CreatedAt:         time.Now(),
			UpdatedAt:         time.Now(),
		},
		{
			ID:                hrReviewID,
			WorkflowID:        workflowID,
			StepOrder:         3,
			StepName:          "hr_review",
			StepLabel:         "HR Document Review",
			ResponsibleRole:   models.RoleHR,
			ActionType:        models.ActionReview,
			TimeoutDays:       5,
			RequiresDocument:  true,
			DocumentType:      "paternity_docs",
			NextStepOnApprove: &hodApprovalID,
			NextStepOnReject:  &requestDocsID,
			CreatedAt:         time.Now(),
			UpdatedAt:         time.Now(),
		},
		{
			ID:                requestDocsID,
			WorkflowID:        workflowID,
			StepOrder:         4,
			StepName:          "request_docs",
			StepLabel:         "Request Document Update",
			ResponsibleRole:   models.RoleHR,
			ActionType:        models.ActionReview,
			TimeoutDays:       7,
			NextStepOnApprove: &hrReviewID,
			CreatedAt:         time.Now(),
			UpdatedAt:         time.Now(),
		},
		{
			ID:                hodApprovalID,
			WorkflowID:        workflowID,
			StepOrder:         5,
			StepName:          "hod_approval",
			StepLabel:         "HOD Approval",
			ResponsibleRole:   models.RoleHOD,
			ActionType:        models.ActionApprove,
			TimeoutDays:       5,
			TimeoutAction:     models.TimeoutEscalate,
			NextStepOnApprove: &approvedStepID,
			CreatedAt:         time.Now(),
			UpdatedAt:         time.Now(),
		},
		{
			ID:              approvedStepID,
			WorkflowID:      workflowID,
			StepOrder:       6,
			StepName:        "approved",
			StepLabel:       "Approved",
			ResponsibleRole: models.RoleHR,
			ActionType:      models.ActionApprove,
			IsTerminal:      true,
			TerminalStatus:  models.StatusApproved,
			NotifyRoles:     models.JSONArray{"manager"},
			CreatedAt:       time.Now(),
			UpdatedAt:       time.Now(),
		},
	}

	if err := s.db.Create(&workflow).Error; err != nil {
		return err
	}
	for _, step := range steps {
		if err := s.db.Create(&step).Error; err != nil {
			return err
		}
	}
	return nil
}

func (s *WorkflowService) seedUnrecordedLeaveWorkflow() error {
	workflowID := uuid.New()
	hodReviewID := uuid.New()
	hrCategorizeID := uuid.New()
	approvedStepID := uuid.New()

	workflow := models.LeaveWorkflow{
		ID:           workflowID,
		LeaveType:    models.LeaveTypeUnrecorded,
		WorkflowName: "Unrecorded Leave Processing",
		Description:  "HOD review, HR categorization as AL or Unpaid",
		FirstStepID:  &hodReviewID,
		IsActive:     true,
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	}

	steps := []models.WorkflowStep{
		{
			ID:                hodReviewID,
			WorkflowID:        workflowID,
			StepOrder:         1,
			StepName:          "hod_review",
			StepLabel:         "HOD Review",
			ResponsibleRole:   models.RoleHOD,
			ActionType:        models.ActionReview,
			TimeoutDays:       5,
			TimeoutAction:     models.TimeoutEscalate,
			NextStepOnApprove: &hrCategorizeID,
			CreatedAt:         time.Now(),
			UpdatedAt:         time.Now(),
		},
		{
			ID:                hrCategorizeID,
			WorkflowID:        workflowID,
			StepOrder:         2,
			StepName:          "hr_categorize",
			StepLabel:         "HR Categorization",
			ResponsibleRole:   models.RoleHR,
			ActionType:        models.ActionCategorize,
			TimeoutDays:       3,
			NextStepOnApprove: &approvedStepID,
			CreatedAt:         time.Now(),
			UpdatedAt:         time.Now(),
		},
		{
			ID:              approvedStepID,
			WorkflowID:      workflowID,
			StepOrder:       3,
			StepName:        "approved",
			StepLabel:       "Categorized & Approved",
			ResponsibleRole: models.RoleHR,
			ActionType:      models.ActionApprove,
			IsTerminal:      true,
			TerminalStatus:  models.StatusApproved,
			NotifyRoles:     models.JSONArray{"manager"},
			CreatedAt:       time.Now(),
			UpdatedAt:       time.Now(),
		},
	}

	if err := s.db.Create(&workflow).Error; err != nil {
		return err
	}
	for _, step := range steps {
		if err := s.db.Create(&step).Error; err != nil {
			return err
		}
	}
	return nil
}

func (s *WorkflowService) seedUnpaidLeaveWorkflow() error {
	workflowID := uuid.New()
	hrReviewID := uuid.New()
	approvedStepID := uuid.New()

	workflow := models.LeaveWorkflow{
		ID:           workflowID,
		LeaveType:    models.LeaveTypeUnpaid,
		WorkflowName: "Unpaid Leave Processing",
		Description:  "Direct HR categorization",
		FirstStepID:  &hrReviewID,
		IsActive:     true,
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	}

	steps := []models.WorkflowStep{
		{
			ID:                hrReviewID,
			WorkflowID:        workflowID,
			StepOrder:         1,
			StepName:          "hr_review",
			StepLabel:         "HR Review",
			ResponsibleRole:   models.RoleHR,
			ActionType:        models.ActionReview,
			TimeoutDays:       3,
			NextStepOnApprove: &approvedStepID,
			CreatedAt:         time.Now(),
			UpdatedAt:         time.Now(),
		},
		{
			ID:              approvedStepID,
			WorkflowID:      workflowID,
			StepOrder:       2,
			StepName:        "approved",
			StepLabel:       "Approved",
			ResponsibleRole: models.RoleHR,
			ActionType:      models.ActionApprove,
			IsTerminal:      true,
			TerminalStatus:  models.StatusApproved,
			NotifyRoles:     models.JSONArray{"manager"},
			CreatedAt:       time.Now(),
			UpdatedAt:       time.Now(),
		},
	}

	if err := s.db.Create(&workflow).Error; err != nil {
		return err
	}
	for _, step := range steps {
		if err := s.db.Create(&step).Error; err != nil {
			return err
		}
	}
	return nil
}
