package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// Department represents an organizational department
type Department struct {
	ID        uuid.UUID  `gorm:"type:uuid;primary_key" json:"id"`
	Name      string     `gorm:"uniqueIndex;not null" json:"name"`
	HODID     *uuid.UUID `json:"hod_id"`
	HOD       *User      `gorm:"foreignKey:HODID;constraint:false" json:"hod,omitempty"`
	CreatedAt time.Time  `json:"created_at"`
	UpdatedAt time.Time  `json:"updated_at"`
}

func (d *Department) BeforeCreate(tx *gorm.DB) error {
	if d.ID == uuid.Nil {
		d.ID = uuid.New()
	}
	return nil
}

// HODDelegation represents a scheduled Acting HOD delegation
type HODDelegation struct {
	ID           uuid.UUID  `gorm:"type:uuid;primary_key" json:"id"`
	DepartmentID uuid.UUID  `gorm:"not null;index" json:"department_id"`
	Department   Department `gorm:"foreignKey:DepartmentID" json:"department,omitempty"`
	DelegatorID  uuid.UUID  `gorm:"not null" json:"delegator_id"` // The HOD who delegates
	Delegator    User       `gorm:"foreignKey:DelegatorID" json:"delegator,omitempty"`
	DelegateID   uuid.UUID  `gorm:"not null" json:"delegate_id"` // The Acting HOD
	Delegate     User       `gorm:"foreignKey:DelegateID" json:"delegate,omitempty"`
	StartDate    time.Time  `gorm:"not null" json:"start_date"`
	EndDate      time.Time  `gorm:"not null" json:"end_date"`
	Reason       string     `json:"reason"`
	IsActive     bool       `gorm:"default:true" json:"is_active"`
	CreatedAt    time.Time  `json:"created_at"`
	UpdatedAt    time.Time  `json:"updated_at"`
}

func (h *HODDelegation) BeforeCreate(tx *gorm.DB) error {
	if h.ID == uuid.Nil {
		h.ID = uuid.New()
	}
	return nil
}
