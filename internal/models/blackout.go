package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// BlackoutDate represents a period where leave applications are restricted
type BlackoutDate struct {
	ID           uuid.UUID   `gorm:"type:uuid;primary_key" json:"id"`
	StartDate    time.Time   `gorm:"not null" json:"start_date"`
	EndDate      time.Time   `gorm:"not null" json:"end_date"`
	Reason       string      `gorm:"not null" json:"reason"`
	ApplyToAll   bool        `gorm:"default:false" json:"apply_to_all"`
	DepartmentID *uuid.UUID  `json:"department_id"`
	Department   *Department `gorm:"foreignKey:DepartmentID;constraint:false" json:"department,omitempty"`
	LeaveType    *LeaveType  `gorm:"type:varchar(20)" json:"leave_type"`
	CreatedBy    uuid.UUID   `gorm:"not null" json:"created_by"`
	Creator      *User       `gorm:"foreignKey:CreatedBy;constraint:false" json:"creator,omitempty"`
	CreatedAt    time.Time   `json:"created_at"`
	UpdatedAt    time.Time   `json:"updated_at"`
}

func (b *BlackoutDate) BeforeCreate(tx *gorm.DB) error {
	if b.ID == uuid.Nil {
		b.ID = uuid.New()
	}
	return nil
}
