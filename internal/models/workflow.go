package models

import (
	"database/sql/driver"
	"encoding/json"
	"errors"
	"time"

	"github.com/google/uuid"
)

// JSONArray for storing string arrays in JSONB
type JSONArray []string

func (j JSONArray) GormDataType() string {
	return "jsonb"
}

func (j JSONArray) Value() (driver.Value, error) {
	return json.Marshal(j)
}

func (j *JSONArray) Scan(value interface{}) error {
	b, ok := value.([]byte)
	if !ok {
		return errors.New("type assertion to []byte failed")
	}
	return json.Unmarshal(b, &j)
}

// WorkflowActionType defines the type of action at a workflow step
type WorkflowActionType string

const (
	ActionApprove    WorkflowActionType = "approve"
	ActionVerify     WorkflowActionType = "verify"
	ActionReview     WorkflowActionType = "review"
	ActionCategorize WorkflowActionType = "categorize"
	ActionSubmit     WorkflowActionType = "submit"
)

// TimeoutAction defines what happens when a step times out
type TimeoutAction string

const (
	TimeoutEscalate    TimeoutAction = "escalate"
	TimeoutAutoApprove TimeoutAction = "auto_approve"
	TimeoutFallback    TimeoutAction = "fallback_step"
	TimeoutConvert     TimeoutAction = "convert_leave_type"
)

// WorkflowStep represents a single step in a leave workflow
type WorkflowStep struct {
	ID                uuid.UUID          `gorm:"type:uuid;primary_key" json:"id"`
	WorkflowID        uuid.UUID          `gorm:"not null;index" json:"workflow_id"`
	StepOrder         int                `gorm:"not null" json:"step_order"`
	StepName          string             `gorm:"not null" json:"step_name"` // e.g., "hod_approval", "hr_review"
	StepLabel         string             `json:"step_label"`                // Human-readable label
	ResponsibleRole   UserRole           `gorm:"type:varchar(20);not null" json:"responsible_role"`
	ActionType        WorkflowActionType `gorm:"type:varchar(20);not null" json:"action_type"`
	TimeoutDays       int                `gorm:"default:7" json:"timeout_days"`
	TimeoutAction     TimeoutAction      `gorm:"type:varchar(30)" json:"timeout_action"`
	FallbackStepID    *uuid.UUID         `json:"fallback_step_id"`
	ConvertToType     *LeaveType         `gorm:"type:varchar(20)" json:"convert_to_type"` // For timeout conversions (AL->EL)
	Conditions        JSONMap            `gorm:"type:jsonb" json:"conditions"`            // Conditional logic
	NextStepOnApprove *uuid.UUID         `json:"next_step_on_approve"`
	NextStepOnReject  *uuid.UUID         `json:"next_step_on_reject"`
	NotifyRoles       JSONArray          `gorm:"type:jsonb" json:"notify_roles"`
	RequiresDocument  bool               `gorm:"default:false" json:"requires_document"`
	DocumentType      string             `json:"document_type"` // e.g., "mc", "justification"
	IsTerminal        bool               `gorm:"default:false" json:"is_terminal"`
	TerminalStatus    LeaveStatus        `gorm:"type:varchar(20)" json:"terminal_status"` // approved/rejected
	CreatedAt         time.Time          `json:"created_at"`
	UpdatedAt         time.Time          `json:"updated_at"`
}

// LeaveWorkflow defines a complete workflow for a leave type
type LeaveWorkflow struct {
	ID           uuid.UUID      `gorm:"type:uuid;primary_key" json:"id"`
	LeaveType    LeaveType      `gorm:"type:varchar(20);not null;index" json:"leave_type"` // Removed unique
	Version      int            `gorm:"default:1" json:"version"`                          // Added Version
	WorkflowName string         `gorm:"not null" json:"workflow_name"`
	Description  string         `json:"description"`
	FirstStepID  *uuid.UUID     `json:"first_step_id"`
	Steps        []WorkflowStep `gorm:"foreignKey:WorkflowID" json:"steps,omitempty"`
	IsActive     bool           `gorm:"default:true" json:"is_active"`
	CreatedAt    time.Time      `json:"created_at"`
	UpdatedAt    time.Time      `json:"updated_at"`
}

// WorkflowStepAction defines what action was taken at a step
type WorkflowStepAction string

const (
	StepActionPending        WorkflowStepAction = "pending"
	StepActionApproved       WorkflowStepAction = "approved"
	StepActionRejected       WorkflowStepAction = "rejected"
	StepActionVerified       WorkflowStepAction = "verified"
	StepActionNotVerified    WorkflowStepAction = "not_verified"
	StepActionRequestedDocs  WorkflowStepAction = "requested_docs"
	StepActionEscalated      WorkflowStepAction = "escalated"
	StepActionCategorizedAL  WorkflowStepAction = "categorized_al"
	StepActionCategorizedUL  WorkflowStepAction = "categorized_unpaid"
	StepActionConvertedToEL  WorkflowStepAction = "converted_to_el"
	StepActionConvertedToUL  WorkflowStepAction = "converted_to_unpaid"
	StepActionTimeoutApplied WorkflowStepAction = "timeout_applied"
	StepActionCancelled      WorkflowStepAction = "cancelled"
)

// LeaveRequestWorkflowState tracks current workflow position for a request
type LeaveRequestWorkflowState struct {
	ID             uuid.UUID          `gorm:"type:uuid;primary_key" json:"id"`
	LeaveRequestID uuid.UUID          `gorm:"unique;not null;index" json:"leave_request_id"`
	WorkflowID     uuid.UUID          `gorm:"not null" json:"workflow_id"`
	CurrentStepID  *uuid.UUID         `json:"current_step_id"`
	CurrentStep    *WorkflowStep      `gorm:"foreignKey:CurrentStepID" json:"current_step,omitempty"`
	StepStartedAt  time.Time          `json:"step_started_at"`
	PreviousStepID *uuid.UUID         `json:"previous_step_id"`
	ActionTaken    WorkflowStepAction `gorm:"type:varchar(30);default:'pending'" json:"action_taken"`
	ActionBy       *uuid.UUID         `json:"action_by"`
	ActionComment  string             `json:"action_comment"`
	StepHistory    JSONMap            `gorm:"type:jsonb" json:"step_history"` // Array of completed steps
	IsComplete     bool               `gorm:"default:false" json:"is_complete"`
	CompletedAt    *time.Time         `json:"completed_at"`
	FinalStatus    LeaveStatus        `gorm:"type:varchar(20)" json:"final_status"`
	CreatedAt      time.Time          `json:"created_at"`
	UpdatedAt      time.Time          `json:"updated_at"`
}

// TableName specifies the table name for GORM
func (WorkflowStep) TableName() string {
	return "workflow_steps"
}

func (LeaveWorkflow) TableName() string {
	return "leave_workflows"
}

func (LeaveRequestWorkflowState) TableName() string {
	return "leave_request_workflow_states"
}
