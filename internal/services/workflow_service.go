package services

import (
	"errors"
	"fmt"
	"leave-management-system/internal/models"
	"strings"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// WorkflowService handles leave workflow operations
type WorkflowService struct {
	db              *gorm.DB
	departmentSvc   *DepartmentService
	delegationSvc   *DelegationService
	notificationSvc *NotificationService
}

func NewWorkflowService(db *gorm.DB, departmentSvc *DepartmentService, delegationSvc *DelegationService) *WorkflowService {
	return &WorkflowService{
		db:            db,
		departmentSvc: departmentSvc,
		delegationSvc: delegationSvc,
	}
}

// SetNotificationService injects the notification service after initialization
func (ws *WorkflowService) SetNotificationService(ns *NotificationService) {
	ws.notificationSvc = ns
}

// GetWorkflowForLeaveType returns the active workflow configuration for a leave type (used at runtime)
func (s *WorkflowService) GetWorkflowForLeaveType(leaveType models.LeaveType) (*models.LeaveWorkflow, error) {
	return s.getWorkflowForLeaveType(s.db, leaveType)
}

func (s *WorkflowService) getWorkflowForLeaveType(db *gorm.DB, leaveType models.LeaveType) (*models.LeaveWorkflow, error) {
	var workflow models.LeaveWorkflow
	// Fetch the active workflow with the highest version
	err := db.Preload("Steps", func(db *gorm.DB) *gorm.DB {
		return db.Order("step_order ASC")
	}).Where("leave_type = ? AND is_active = ?", leaveType, true).
		Order("version DESC"). // Get highest version
		First(&workflow).Error
	if err != nil {
		return nil, err
	}
	return &workflow, nil
}

// GetWorkflowByLeaveType returns the latest workflow for a leave type regardless of active status (used by admin)
func (s *WorkflowService) GetWorkflowByLeaveType(leaveType models.LeaveType) (*models.LeaveWorkflow, error) {
	var workflow models.LeaveWorkflow
	// Fetch the latest version (active or not)
	err := s.db.Preload("Steps", func(db *gorm.DB) *gorm.DB {
		return db.Order("step_order ASC")
	}).Where("leave_type = ?", leaveType).
		Order("version DESC").
		First(&workflow).Error
	if err != nil {
		return nil, err
	}
	return &workflow, nil
}

// GetAllWorkflows returns all workflow configurations (latest versions)
func (s *WorkflowService) GetAllWorkflows() ([]models.LeaveWorkflow, error) {
	var workflows []models.LeaveWorkflow
	// This is tricky with versioning. We want one per leave type.
	// Since there are few leave types, we can fetch all and filter or use distinct on.
	// SQLite supports distinct on, but let's be generic.
	// Simplified: Fetch all IsActive=true.
	err := s.db.Preload("Steps", func(db *gorm.DB) *gorm.DB {
		return db.Order("step_order ASC")
	}).Where("is_active = ?", true).Find(&workflows).Error
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
	return s.InitializeWorkflowStateWithTx(s.db, leaveRequest)
}

func (s *WorkflowService) InitializeWorkflowStateWithTx(tx *gorm.DB, leaveRequest *models.LeaveRequest) (*models.LeaveRequestWorkflowState, error) {
	workflow, err := s.getWorkflowForLeaveType(tx, leaveRequest.LeaveType)
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

	if err := tx.Create(state).Error; err != nil {
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

	// [UNIFIED CHRONOLOGY] Also create a persistent chronology entry for the main history view
	chronology := models.Chronology{
		ID:             uuid.New(),
		LeaveRequestID: leaveRequestID,
		Action:         string(action),
		ActorID:        actorID,
		Comment:        comment,
		Metadata: models.JSONMap{
			"step_name":  currentStep.StepName,
			"step_label": currentStep.StepLabel,
			"action":     string(action),
		},
		CreatedAt: time.Now(),
	}
	if err := tx.Create(&chronology).Error; err != nil {
		return nil, err
	}

	// Fetch full request to check IsNegativeBalance
	var leaveRequest models.LeaveRequest
	tx.First(&leaveRequest, "id = ?", leaveRequestID)

	// Determine next step based on action
	var nextStepID *uuid.UUID
	var isTerminal bool
	var terminalStatus models.LeaveStatus

	// Helper function to resolve dynamic skips
	resolveNextStep := func(initialNextStepID *uuid.UUID) *uuid.UUID {
		currentCheckID := initialNextStepID
		for currentCheckID != nil {
			var nextStep models.WorkflowStep
			if err := tx.First(&nextStep, "id = ?", currentCheckID).Error; err != nil {
				break
			}

			// If this step requires HR ONLY IF negative, and request is NOT negative, skip it
			if requiresHR, ok := nextStep.Conditions["requires_hr_if_negative"].(bool); ok && requiresHR {
				if !leaveRequest.IsNegativeBalance {
					// We need to skip this step. What's its next step?
					// Assume it acts as an auto-approve bypass
					if nextStep.IsTerminal {
						isTerminal = true
						terminalStatus = nextStep.TerminalStatus
						return nil // End of workflow
					}
					currentCheckID = nextStep.NextStepOnApprove
					continue
				}
			}
			break // Step is valid, no bypass needed
		}
		return currentCheckID
	}

	switch action {
	case models.StepActionApproved, models.StepActionVerified:
		nextStepID = resolveNextStep(currentStep.NextStepOnApprove)
		// If current step is terminal and it didn't bypass to a new step
		if currentStep.IsTerminal && nextStepID == nil && !isTerminal {
			isTerminal = true
			terminalStatus = currentStep.TerminalStatus
		}
	case models.StepActionRejected, models.StepActionNotVerified:
		nextStepID = resolveNextStep(currentStep.NextStepOnReject)
		// If current step rejects directly, or it bypassed to the end without finding a next step
		if currentStep.NextStepOnReject == nil && !isTerminal {
			isTerminal = true
			terminalStatus = models.StatusRejected
		}
	case models.StepActionCategorizedAL, models.StepActionCategorizedUL:
		// For categorization steps, move to next step
		nextStepID = resolveNextStep(currentStep.NextStepOnApprove)
		if currentStep.IsTerminal && nextStepID == nil && !isTerminal {
			isTerminal = true
			terminalStatus = models.StatusApproved
		}
	case models.StepActionRequestedDocs:
		// Stay at current step, waiting for document resubmission.
		// Set a flag so the terminal check below does NOT close the workflow.
		state.ActionTaken = models.StepActionRequestedDocs
		state.StepStartedAt = time.Now()
		// Explicitly set nextStepID to current step to prevent the nil-termination logic from firing
		nextStepID = state.CurrentStepID
	case models.StepActionEscalated:
		// Move to fallback step if configured
		if currentStep.FallbackStepID != nil {
			nextStepID = resolveNextStep(currentStep.FallbackStepID)
		} else {
			isTerminal = true
			terminalStatus = models.StatusEscalated
		}
	default:
		return nil, fmt.Errorf("unknown action: %s", action)
	}

	if !isTerminal && nextStepID != nil {
		var nextStep models.WorkflowStep
		if err := tx.First(&nextStep, "id = ?", *nextStepID).Error; err == nil && isTerminalEndpointStep(&nextStep) {
			isTerminal = true
			terminalStatus = nextStep.TerminalStatus
			nextStepID = nil
		}
	}

	// Update state
	if isTerminal || nextStepID == nil {
		state.IsComplete = true
		now := time.Now()
		state.CompletedAt = &now

		// If it's the end of the line and no terminal status was explicitly defined, infer from action
		if !isTerminal {
			switch action {
			case models.StepActionApproved, models.StepActionVerified, models.StepActionCategorizedAL, models.StepActionCategorizedUL:
				terminalStatus = models.StatusApproved
			case models.StepActionRejected, models.StepActionNotVerified:
				terminalStatus = models.StatusRejected
			case models.StepActionEscalated:
				terminalStatus = models.StatusEscalated
			// Stay pending for RequestedDocs if we somehow reach here, though nextStepID wouldn't be nil normally
			default:
				terminalStatus = models.StatusApproved // Failsafe
			}
		}

		state.FinalStatus = terminalStatus
		state.CurrentStepID = nil
	} else {
		state.PreviousStepID = state.CurrentStepID
		state.CurrentStepID = nextStepID
		state.StepStartedAt = time.Now()
		// Only reset to pending if this isn't a "stay at current step" action like RequestedDocs
		if action != models.StepActionRequestedDocs {
			state.ActionTaken = models.StepActionPending
		}
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
			if err := s.db.Save(state).Error; err != nil {
				return state.LeaveRequestID, err
			}
			// Record chronology entry for the timeout fallback
			chronology := models.Chronology{
				ID:             uuid.New(),
				LeaveRequestID: state.LeaveRequestID,
				Action:         string(models.StepActionTimeoutApplied),
				ActorID:        uuid.Nil,
				Comment:        fmt.Sprintf("Step '%s' timed out after %d days. Escalated to fallback step.", step.StepName, step.TimeoutDays),
				Metadata: models.JSONMap{
					"step_name":      step.StepName,
					"timeout_action": string(step.TimeoutAction),
					"timeout_days":   step.TimeoutDays,
				},
				CreatedAt: time.Now(),
			}
			if err := s.db.Create(&chronology).Error; err != nil {
				return state.LeaveRequestID, err
			}
			return state.LeaveRequestID, nil
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

		// Re-initialize workflow for new leave type (use tx to read within the same transaction)
		newWorkflow, err := s.getWorkflowForLeaveType(tx, newType)
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

	// Helper to fetch admins
	getAdmins := func() ([]models.User, error) {
		var admins []models.User
		err := s.db.Preload("RoleRef").
			Joins("JOIN roles ON roles.id = users.role_id").
			Where("roles.name IN ? AND users.is_active = ?", []models.UserRole{models.RoleAdmin, models.RoleSysAdmin}, true).
			Find(&admins).Error
		return admins, err
	}

	// 1. Handle HOD Role (Department Logic)
	if state.CurrentStep.ResponsibleRole == models.RoleHOD && s.departmentSvc != nil {
		// Get the leave request to find the applicant
		var leaveRequest models.LeaveRequest
		if err := s.db.Preload("User.RoleRef").First(&leaveRequest, "id = ?", state.LeaveRequestID).Error; err != nil {
			return nil, fmt.Errorf("failed to load leave request: %w", err)
		}

		// [SMART ROUTING] HOD Self-Approval Bypass
		// If the applicant IS the HOD (or acting HOD), they cannot approve themselves.
		// Route to their Reporting Manager instead.
		if leaveRequest.User.Role == models.RoleHOD {
			// If they have a direct manager, route to them
			if leaveRequest.User.ManagerID != nil {
				// We reuse the Manager Logic below essentially, but let's be explicit
				// Use DelegationService if available to check if THAT manager delegated
				if s.delegationSvc != nil {
					delegate, err := s.delegationSvc.GetActiveDelegation(*leaveRequest.User.ManagerID, time.Now())
					if err == nil && delegate != nil {
						return []models.User{*delegate}, nil
					}
				}
				// Fallback to the actual manager
				var manager models.User
				if err := s.db.First(&manager, "id = ?", *leaveRequest.User.ManagerID).Error; err == nil {
					return []models.User{manager}, nil
				}
			}
			// If HOD has no manager (Top Level HOD?), Fallback to Admin
			return getAdmins()
		}

		// Normal Staff: Route to Department HOD
		if leaveRequest.User.DepartmentID != nil {
			approver, err := s.departmentSvc.ResolveApproverForDepartment(*leaveRequest.User.DepartmentID)
			if err == nil && approver != nil {
				// [SMART ROUTING] Final Self-Approval Guard
				// Even if resolved, ensure it's not the applicant (e.g. data inconsistency)
				if approver.ID == leaveRequest.UserID {
					return getAdmins()
				}
				return []models.User{*approver}, nil
			}
			// If resolution fails, fall through to role-based lookup
		}
	}

	// 2. Handle Manager Role (Direct Supervisor Logic)
	if state.CurrentStep.ResponsibleRole == models.RoleManager {
		var leaveRequest models.LeaveRequest
		// Need to preload User and User.Manager
		if err := s.db.Preload("User").Preload("User.Manager").First(&leaveRequest, "id = ?", state.LeaveRequestID).Error; err == nil {

			// If user has a manager
			if leaveRequest.User.ManagerID != nil {
				// Check for Delegation first
				if s.delegationSvc != nil {
					delegate, err := s.delegationSvc.GetActiveDelegation(*leaveRequest.User.ManagerID, time.Now())
					if err == nil && delegate != nil {
						// Ensure delegate is not the applicant (unlikely but possible)
						if delegate.ID != leaveRequest.User.ID {
							return []models.User{*delegate}, nil
						}
					}
				}

				// Return the specific manager
				if leaveRequest.User.Manager != nil {
					// [SMART ROUTING] Self-Check
					if leaveRequest.User.Manager.ID == leaveRequest.User.ID {
						// Should not happen in healthy DB, but safe guard -> Admin
						return getAdmins()
					}
					return []models.User{*leaveRequest.User.Manager}, nil
				}
			} else {
				// [SMART ROUTING] Orphan Manager Fallback
				// Applicant has NO Manager (Top of hierarchy, e.g. CEO/Director)
				// Do NOT fall through to "All Managers". Route to Admin.
				return getAdmins()
			}
		}
	}

	// 3. Default: Role-based lookup (HR, Admin, etc.)
	var users []models.User
	err := s.db.Preload("RoleRef").
		Joins("JOIN roles ON roles.id = users.role_id").
		Where("roles.name = ? AND users.is_active = ?", state.CurrentStep.ResponsibleRole, true).
		Find(&users).Error

	// [SMART ROUTING] Filter out Applicant from Role-Based Pool
	if err == nil {
		var validUsers []models.User
		requestsSelf := false

		// [OPTIMIZATION] Fetch the applicant ID ONCE outside the loop to prevent N+1 queries
		var lr models.LeaveRequest
		s.db.Select("user_id").First(&lr, "id = ?", state.LeaveRequestID)

		for _, u := range users {
			if u.ID != lr.UserID {
				validUsers = append(validUsers, u)
			} else {
				requestsSelf = true
			}
		}

		// If we filtered everyone out (e.g. the only HR is the applicant), fallback to Admin
		if len(validUsers) == 0 && requestsSelf {
			return getAdmins()
		}

		return validUsers, nil
	}

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

// EnsureEditableWorkflow checks if the workflow is in use. If so, it creates a new version.
// Returns: newWorkflowID, map[oldStepID]newStepID, error
func (s *WorkflowService) EnsureEditableWorkflow(workflowID uuid.UUID) (uuid.UUID, map[uuid.UUID]uuid.UUID, error) {
	var workflow models.LeaveWorkflow
	if err := s.db.Preload("Steps").First(&workflow, "id = ?", workflowID).Error; err != nil {
		return uuid.Nil, nil, err
	}

	// Check if any requests are using this workflow ID
	var count int64
	s.db.Model(&models.LeaveRequestWorkflowState{}).Where("workflow_id = ?", workflowID).Count(&count)

	if count == 0 {
		return workflowID, nil, nil // Safe to edit in place
	}

	// In use: Create new version
	newWorkflowID := uuid.New()
	newWorkflow := models.LeaveWorkflow{
		ID:           newWorkflowID,
		LeaveType:    workflow.LeaveType,
		Version:      workflow.Version + 1,
		WorkflowName: workflow.WorkflowName,
		Description:  workflow.Description,
		IsActive:     true,
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	}

	stepMap := make(map[uuid.UUID]uuid.UUID)

	err := s.db.Transaction(func(tx *gorm.DB) error {
		// 1. Archive old workflow
		if err := tx.Model(&models.LeaveWorkflow{}).Where("id = ?", workflowID).Update("is_active", false).Error; err != nil {
			return err
		}

		// 2. Create new workflow
		if err := tx.Create(&newWorkflow).Error; err != nil {
			return err
		}

		// 3. Clone steps
		var newSteps []models.WorkflowStep
		for _, step := range workflow.Steps {
			newStepID := uuid.New()
			stepMap[step.ID] = newStepID

			newStep := step
			newStep.ID = newStepID
			newStep.WorkflowID = newWorkflowID
			newStep.CreatedAt = time.Now()
			newStep.UpdatedAt = time.Now()
			newSteps = append(newSteps, newStep)
		}

		// 4. Update links
		for i := range newSteps {
			if newSteps[i].NextStepOnApprove != nil {
				if newID, ok := stepMap[*newSteps[i].NextStepOnApprove]; ok {
					newSteps[i].NextStepOnApprove = &newID
				}
			}
			if newSteps[i].NextStepOnReject != nil {
				if newID, ok := stepMap[*newSteps[i].NextStepOnReject]; ok {
					newSteps[i].NextStepOnReject = &newID
				}
			}
			if newSteps[i].FallbackStepID != nil {
				if newID, ok := stepMap[*newSteps[i].FallbackStepID]; ok {
					newSteps[i].FallbackStepID = &newID
				}
			}
		}

		// 5. Save steps
		if len(newSteps) > 0 {
			if err := tx.Create(&newSteps).Error; err != nil {
				return err
			}
		}

		// 6. Update FirstStepID
		if workflow.FirstStepID != nil {
			if newFirstID, ok := stepMap[*workflow.FirstStepID]; ok {
				if err := tx.Model(&newWorkflow).Update("first_step_id", newFirstID).Error; err != nil {
					return err
				}
			}
		}

		return nil
	})

	return newWorkflowID, stepMap, err
}

// UpdateWorkflow updates a workflow configuration (Versioning Aware)
func (s *WorkflowService) UpdateWorkflow(workflow *models.LeaveWorkflow) error {
	targetID, _, err := s.EnsureEditableWorkflow(workflow.ID)
	if err != nil {
		return err
	}

	if targetID != workflow.ID {
		workflow.ID = targetID
		workflow.Version = workflow.Version + 1
	}

	workflow.UpdatedAt = time.Now()
	return s.db.Save(workflow).Error
}

// CreateWorkflowStep creates a new step in a workflow
func (s *WorkflowService) CreateWorkflowStep(step *models.WorkflowStep) error {
	// Ensure we are editing a safe version
	targetWorkflowID, stepMap, err := s.EnsureEditableWorkflow(step.WorkflowID)
	if err != nil {
		return err
	}

	// Update the step to point to the correct workflow (could be new version)
	step.WorkflowID = targetWorkflowID
	remapWorkflowStepLinks(step, stepMap)

	step.ID = uuid.New()
	step.CreatedAt = time.Now()
	step.UpdatedAt = time.Now()
	return s.db.Create(step).Error
}

// UpdateWorkflowStep updates a workflow step
func (s *WorkflowService) UpdateWorkflowStep(step *models.WorkflowStep) error {
	// 1. Get current state of the step to find WorkflowID
	var existingStep models.WorkflowStep
	if err := s.db.First(&existingStep, "id = ?", step.ID).Error; err != nil {
		return err
	}

	// 2. Ensure Workflow is editable
	targetWorkflowID, stepMap, err := s.EnsureEditableWorkflow(existingStep.WorkflowID)
	if err != nil {
		return err
	}

	// 3. If workflow was cloned, we need to find the NEW step ID corresponding to 'step.ID'
	if targetWorkflowID != existingStep.WorkflowID {
		newStepID, found := stepMap[step.ID]
		if !found {
			return errors.New("failed to resolve step ID in new workflow version")
		}
		// Redirect update to the new step
		step.ID = newStepID
		step.WorkflowID = targetWorkflowID
	}

	step.UpdatedAt = time.Now()
	return s.db.Save(step).Error
}

// DeleteWorkflowStep removes a step from a workflow
func (s *WorkflowService) DeleteWorkflowStep(stepID uuid.UUID) error {
	// 1. Get step
	var step models.WorkflowStep
	if err := s.db.First(&step, "id = ?", stepID).Error; err != nil {
		return err
	}

	// 2. Ensure editable
	targetWorkflowID, stepMap, err := s.EnsureEditableWorkflow(step.WorkflowID)
	if err != nil {
		return err
	}

	// 3. Target correct step
	targetStepID := stepID
	if targetWorkflowID != step.WorkflowID {
		if newID, ok := stepMap[stepID]; ok {
			targetStepID = newID
		} else {
			return errors.New("failed to resolve step ID for deletion in new version")
		}
	}

	return s.db.Delete(&models.WorkflowStep{}, "id = ?", targetStepID).Error
}

// ReorderWorkflowSteps updates the order of steps in a workflow
func (s *WorkflowService) ReorderWorkflowSteps(workflowID uuid.UUID, stepOrders map[uuid.UUID]int) error {
	// 1. Ensure editable
	targetWorkflowID, stepMap, err := s.EnsureEditableWorkflow(workflowID)
	if err != nil {
		return err
	}

	return s.db.Transaction(func(tx *gorm.DB) error {
		for stepID, order := range stepOrders {
			// Resolve ID
			targetStepID := stepID
			if targetWorkflowID != workflowID {
				if newID, ok := stepMap[stepID]; ok {
					targetStepID = newID
				} else {
					// If mapped step not found (maybe deleted?), skip or error.
					// Safer to skip if partial map?
					// But stepOrders usually comes from current view.
					// If we cloned, stepMap should have it.
					continue
				}
			}

			if err := tx.Model(&models.WorkflowStep{}).
				Where("id = ? AND workflow_id = ?", targetStepID, targetWorkflowID).
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

	// Check negative balance requirement
	if requiresHR, ok := step.Conditions["requires_hr_if_negative"].(bool); ok && requiresHR {
		if step.ResponsibleRole == models.RoleHR && !request.IsNegativeBalance {
			// If this step is for HR but the balance is NOT negative, we can auto-skip this condition
			// But EvaluateConditions usually strictly blocks progression, so returning false means "Condition not met, cannot approve".
			// Wait, evaluated conditions are usually for stopping progression (e.g. min advance, require docs).
			// If we want skipping, EvaluateConditions is not the right place. We should implement bypass logic in ProcessActionWithTx instead.
		}
	}

	return true, ""
}

func isTerminalEndpointStep(step *models.WorkflowStep) bool {
	if step == nil || !step.IsTerminal {
		return false
	}

	if step.NextStepOnApprove != nil || step.NextStepOnReject != nil || step.FallbackStepID != nil {
		return false
	}

	stepName := strings.ToLower(step.StepName)
	return stepName == "approved" || stepName == "rejected" || stepName == "convert_unpaid"
}

func remapWorkflowStepLinks(step *models.WorkflowStep, stepMap map[uuid.UUID]uuid.UUID) {
	if step == nil || len(stepMap) == 0 {
		return
	}

	if step.NextStepOnApprove != nil {
		if newID, ok := stepMap[*step.NextStepOnApprove]; ok {
			step.NextStepOnApprove = &newID
		}
	}

	if step.NextStepOnReject != nil {
		if newID, ok := stepMap[*step.NextStepOnReject]; ok {
			step.NextStepOnReject = &newID
		}
	}

	if step.FallbackStepID != nil {
		if newID, ok := stepMap[*step.FallbackStepID]; ok {
			step.FallbackStepID = &newID
		}
	}
}

// FixTerminalStepEndpoints is a one-time migration that removes dummy terminal
// endpoint steps (e.g., "approved" steps that just sit there waiting for another click).
// It makes the last real action step terminal instead, eliminating double-approve bugs.
func (s *WorkflowService) FixTerminalStepEndpoints() error {
	// Find all dummy terminal steps: steps that are terminal, have no next steps,
	// and are named generically (approved, rejected, etc.)
	var dummyTerminals []models.WorkflowStep
	err := s.db.Where(
		"is_terminal = ? AND next_step_on_approve IS NULL AND next_step_on_reject IS NULL AND step_name IN ?",
		true,
		[]string{"approved", "rejected", "convert_unpaid"},
	).Find(&dummyTerminals).Error
	if err != nil || len(dummyTerminals) == 0 {
		return nil // Nothing to fix
	}

	return s.db.Transaction(func(tx *gorm.DB) error {
		for _, dummy := range dummyTerminals {
			// Find parent steps that point to this dummy via NextStepOnApprove
			var parentSteps []models.WorkflowStep
			tx.Where("next_step_on_approve = ?", dummy.ID).Find(&parentSteps)
			for _, parent := range parentSteps {
				parent.IsTerminal = true
				parent.TerminalStatus = dummy.TerminalStatus
				parent.NextStepOnApprove = nil
				if err := tx.Save(&parent).Error; err != nil {
					return err
				}
			}

			// Find parent steps that point to this dummy via NextStepOnReject
			var rejectParents []models.WorkflowStep
			tx.Where("next_step_on_reject = ?", dummy.ID).Find(&rejectParents)
			for _, parent := range rejectParents {
				// For reject parents, only clear the pointer (rejection is already handled)
				parent.NextStepOnReject = nil
				if err := tx.Save(&parent).Error; err != nil {
					return err
				}
			}

			// Also fix any active workflow states stuck at dummy steps
			tx.Model(&models.LeaveRequestWorkflowState{}).
				Where("current_step_id = ? AND is_complete = ?", dummy.ID, false).
				Updates(map[string]interface{}{
					"is_complete":     true,
					"final_status":    dummy.TerminalStatus,
					"current_step_id": nil,
					"completed_at":    time.Now(),
					"updated_at":      time.Now(),
				})

			// Delete the dummy step
			if err := tx.Delete(&dummy).Error; err != nil {
				return err
			}
		}
		return nil
	})
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
	hrNegativeReviewID := uuid.New()
	hrFallbackStepID := uuid.New()
	approvedStepID := uuid.New()

	workflow := models.LeaveWorkflow{
		ID:           workflowID,
		LeaveType:    models.LeaveTypeAnnual,
		WorkflowName: "Annual Leave Approval",
		Description:  "HOD approval. Requires HR review if balance is negative.",
		FirstStepID:  &hodStepID,
		IsActive:     true,
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	}

	steps := []models.WorkflowStep{
		{
			ID:                hodStepID,
			WorkflowID:        workflowID,
			StepOrder:         1,
			StepName:          "hod_approval",
			StepLabel:         "HOD Approval",
			ResponsibleRole:   models.RoleHOD,
			ActionType:        models.ActionApprove,
			TimeoutDays:       7,
			TimeoutAction:     models.TimeoutFallback,
			FallbackStepID:    &hrFallbackStepID,
			NextStepOnApprove: &hrNegativeReviewID,
			NotifyRoles:       models.JSONArray{"manager"},
			CreatedAt:         time.Now(),
			UpdatedAt:         time.Now(),
		},
		{
			ID:                hrNegativeReviewID,
			WorkflowID:        workflowID,
			StepOrder:         2,
			StepName:          "hr_negative_balance_review",
			StepLabel:         "HR Negative Balance Review",
			ResponsibleRole:   models.RoleHR,
			ActionType:        models.ActionReview,
			TimeoutDays:       7,
			NextStepOnApprove: &approvedStepID,
			Conditions:        models.JSONMap{"requires_hr_if_negative": true},
			NotifyRoles:       models.JSONArray{"manager"},
			CreatedAt:         time.Now(),
			UpdatedAt:         time.Now(),
		},
		{
			ID:                hrFallbackStepID,
			WorkflowID:        workflowID,
			StepOrder:         3,
			StepName:          "hr_fallback",
			StepLabel:         "HR Decision (HOD Timeout)",
			ResponsibleRole:   models.RoleHR,
			ActionType:        models.ActionApprove,
			TimeoutDays:       3, // Optional: give HR 3 days
			NextStepOnApprove: &hrNegativeReviewID,
			NotifyRoles:       models.JSONArray{"manager"},
			CreatedAt:         time.Now(),
			UpdatedAt:         time.Now(),
		},
		{
			ID:              approvedStepID,
			WorkflowID:      workflowID,
			StepOrder:       4,
			StepName:        "approved",
			StepLabel:       "Approved",
			ResponsibleRole: models.RoleHR, // Just a terminal role label
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
	hrFallbackStepID := uuid.New()
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
			ID:                hodStepID,
			WorkflowID:        workflowID,
			StepOrder:         1,
			StepName:          "hod_approval",
			StepLabel:         "HOD Approval",
			ResponsibleRole:   models.RoleHOD,
			ActionType:        models.ActionApprove,
			TimeoutDays:       7,
			TimeoutAction:     models.TimeoutFallback,
			FallbackStepID:    &hrFallbackStepID,
			NextStepOnApprove: &hrStepID,
			NotifyRoles:       models.JSONArray{"manager"},
			CreatedAt:         time.Now(),
			UpdatedAt:         time.Now(),
		},
		{
			ID:                hrFallbackStepID,
			WorkflowID:        workflowID,
			StepOrder:         2,
			StepName:          "hr_fallback",
			StepLabel:         "HR Decision (HOD Timeout)",
			ResponsibleRole:   models.RoleHR,
			ActionType:        models.ActionApprove,
			TimeoutDays:       3,
			NextStepOnApprove: &hrStepID,
			NotifyRoles:       models.JSONArray{"manager"},
			CreatedAt:         time.Now(),
			UpdatedAt:         time.Now(),
		},
		{
			ID:                hrStepID,
			WorkflowID:        workflowID,
			StepOrder:         3,
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
			StepOrder:       4,
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
			StepOrder:       5,
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
