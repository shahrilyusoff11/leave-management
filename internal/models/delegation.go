package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// UserDelegation represents a general "Acting Manager" delegation
// Allows any user (usually a manager) to delegate their approval duties to another user
type UserDelegation struct {
	ID          uuid.UUID `gorm:"type:uuid;primary_key" json:"id"`
	DelegatorID uuid.UUID `gorm:"not null;index" json:"delegator_id"` // The manager who is away
	Delegator   *User     `gorm:"foreignKey:DelegatorID" json:"delegator,omitempty"`
	DelegateID  uuid.UUID `gorm:"not null;index" json:"delegate_id"` // The person taking over
	Delegate    *User     `gorm:"foreignKey:DelegateID" json:"delegate,omitempty"`
	StartDate   time.Time `gorm:"not null;index" json:"start_date"`
	EndDate     time.Time `gorm:"not null;index" json:"end_date"`
	Status      string    `gorm:"default:'active'" json:"status"` // "active", "cancelled"
	Reason      string    `json:"reason"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

func (d *UserDelegation) BeforeCreate(tx *gorm.DB) error {
	if d.ID == uuid.Nil {
		d.ID = uuid.New()
	}
	return nil
}
