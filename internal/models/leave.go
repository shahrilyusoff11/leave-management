package models

import (
	"database/sql/driver"
	"encoding/json"
	"errors"
	"time"

	"github.com/google/uuid"
)

type LeaveType string

const (
	LeaveTypeAnnual          LeaveType = "annual"
	LeaveTypeSick            LeaveType = "sick"
	LeaveTypeMaternity       LeaveType = "maternity"
	LeaveTypePaternity       LeaveType = "paternity"
	LeaveTypeEmergency       LeaveType = "emergency"
	LeaveTypeUnpaid          LeaveType = "unpaid"
	LeaveTypeUnrecorded      LeaveType = "unrecorded"
	LeaveTypeHospitalization LeaveType = "hospitalization"
)

type LeaveStatus string

const (
	StatusPending   LeaveStatus = "pending"
	StatusApproved  LeaveStatus = "approved"
	StatusRejected  LeaveStatus = "rejected"
	StatusCancelled LeaveStatus = "cancelled"
	StatusEscalated LeaveStatus = "escalated"
)

type LeaveRequest struct {
	ID                     uuid.UUID    `gorm:"type:uuid;primary_key" json:"id"`
	UserID                 uuid.UUID    `gorm:"not null" json:"user_id"`
	User                   User         `gorm:"foreignKey:UserID" json:"user,omitempty"`
	LeaveType              LeaveType    `gorm:"type:varchar(20);not null" json:"leave_type"`
	StartDate              time.Time    `gorm:"not null" json:"start_date"`
	EndDate                time.Time    `gorm:"not null" json:"end_date"`
	DurationDays           float64      `gorm:"not null" json:"duration_days"` // Float for half-day leaves
	Reason                 string       `json:"reason"`
	Status                 LeaveStatus  `gorm:"type:varchar(20);default:'pending'" json:"status"`
	ApproverID             *uuid.UUID   `json:"approver_id"`
	Approver               *User        `gorm:"foreignKey:ApproverID" json:"approver,omitempty"`
	ApprovedAt             *time.Time   `json:"approved_at"`
	RejectedAt             *time.Time   `json:"rejected_at"`
	RejectionReason        string       `json:"rejection_reason"`
	AttachmentURL          string       `json:"attachment_url"`
	AttachmentFileName     string       `json:"attachment_file_name"`
	IsEscalated            bool         `gorm:"default:false" json:"is_escalated"`
	EscalatedAt            *time.Time   `json:"escalated_at"`
	UnrecordedLeaveSubtype string       `json:"unrecorded_leave_subtype"` // For marriage, compassionate, hajj
	ChronologyEntries      []Chronology `gorm:"foreignKey:LeaveRequestID" json:"chronology_entries,omitempty"`
	// Workflow fields
	WorkflowStateID *uuid.UUID                 `json:"workflow_state_id"`
	WorkflowState   *LeaveRequestWorkflowState `gorm:"foreignKey:WorkflowStateID" json:"workflow_state,omitempty"`
	ConvertedFrom   *LeaveType                 `gorm:"type:varchar(20)" json:"converted_from"` // Track AL→EL or EL→Unpaid conversions
	CreatedAt       time.Time                  `json:"created_at"`
	UpdatedAt       time.Time                  `json:"updated_at"`
}

type LeaveBalance struct {
	ID               uuid.UUID `gorm:"type:uuid;primary_key" json:"id"`
	UserID           uuid.UUID `gorm:"not null;index" json:"user_id"`
	LeaveType        LeaveType `gorm:"type:varchar(20);not null" json:"leave_type"`
	Year             int       `gorm:"not null" json:"year"`
	TotalEntitlement float64   `gorm:"not null" json:"total_entitlement"`
	Used             float64   `gorm:"not null;default:0" json:"used"`
	CarriedForward   float64   `gorm:"not null;default:0" json:"carried_forward"`
	Adjusted         float64   `gorm:"not null;default:0" json:"adjusted"` // Manual adjustments by HR
	IsOverridden     bool      `gorm:"default:false" json:"is_overridden"` // HR override flag
	CreatedAt        time.Time `json:"created_at"`
	UpdatedAt        time.Time `json:"updated_at"`
}

type Chronology struct {
	ID             uuid.UUID `gorm:"type:uuid;primary_key" json:"id"`
	LeaveRequestID uuid.UUID `gorm:"not null;index" json:"leave_request_id"`
	Action         string    `gorm:"not null" json:"action"` // submitted, modified, commented, approved, rejected
	ActorID        uuid.UUID `gorm:"not null" json:"actor_id"`
	Actor          User      `gorm:"foreignKey:ActorID" json:"actor"`
	Comment        string    `json:"comment"`
	Metadata       JSONMap   `gorm:"type:jsonb" json:"metadata"` // Store before/after states
	CreatedAt      time.Time `json:"created_at"`
}

type PublicHoliday struct {
	ID          uuid.UUID `gorm:"type:uuid;primary_key" json:"id"`
	Name        string    `gorm:"not null" json:"name"`
	Date        time.Time `gorm:"not null" json:"date"`
	Description string    `json:"description"`
	State       string    `json:"state"` // Empty for nationwide
	IsActive    bool      `gorm:"default:true" json:"is_active"`
	CreatedAt   time.Time `json:"created_at"`
}

// LeaveTypeConfig stores configurable settings for each leave type
type LeaveTypeConfig struct {
	ID                    uuid.UUID `gorm:"type:uuid;primary_key" json:"id"`
	LeaveType             LeaveType `gorm:"type:varchar(20);unique;not null" json:"leave_type"`
	BaseEntitlement       float64   `gorm:"not null;default:0" json:"base_entitlement"`
	YearsOfServiceTiers   JSONMap   `gorm:"type:jsonb" json:"years_of_service_tiers"` // {"2": 2, "5": 4, "10": 6}
	ProrateFirstYear      bool      `gorm:"default:true" json:"prorate_first_year"`
	AllowCarryForward     bool      `gorm:"default:false" json:"allow_carry_forward"`
	MaxCarryForwardDays   int       `gorm:"default:0" json:"max_carry_forward_days"`
	MaxDaysPerApplication *int      `json:"max_days_per_application"`
	RequiresAttachment    bool      `gorm:"default:false" json:"requires_attachment"`
	MinAdvanceDays        int       `gorm:"default:0" json:"min_advance_days"`
	IsActive              bool      `gorm:"default:true" json:"is_active"`
	DisplayOrder          int       `gorm:"default:0" json:"display_order"`
	CreatedAt             time.Time `json:"created_at"`
	UpdatedAt             time.Time `json:"updated_at"`
}

type AuditLog struct {
	ID          uuid.UUID `gorm:"type:uuid;primary_key" json:"id"`
	ActorID     uuid.UUID `gorm:"not null" json:"actor_id"`
	ActorEmail  string    `gorm:"not null" json:"actor_email"`
	ActorRole   UserRole  `gorm:"type:varchar(20)" json:"actor_role"`
	Action      string    `gorm:"not null" json:"action"`
	TargetID    uuid.UUID `json:"target_id"`
	TargetType  string    `json:"target_type"` // user, leave_request, leave_balance
	BeforeState JSONMap   `gorm:"type:jsonb" json:"before_state"`
	AfterState  JSONMap   `gorm:"type:jsonb" json:"after_state"`
	IPAddress   string    `json:"ip_address"`
	UserAgent   string    `json:"user_agent"`
	Method      string    `json:"method"`
	Endpoint    string    `json:"endpoint"`
	CreatedAt   time.Time `json:"created_at"`
}

// JSONMap for storing JSON in database
type JSONMap map[string]interface{}

func (j JSONMap) GormDataType() string {
	return "jsonb"
}

func (j JSONMap) Value() (driver.Value, error) {
	return json.Marshal(j)
}

func (j *JSONMap) Scan(value interface{}) error {
	b, ok := value.([]byte)
	if !ok {
		return errors.New("type assertion to []byte failed")
	}
	return json.Unmarshal(b, &j)
}
