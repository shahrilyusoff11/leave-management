package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// NotificationType defines the category of the notification
type NotificationType string

const (
	NotificationTypeLeave    NotificationType = "leave"
	NotificationTypeWorkflow NotificationType = "workflow"
	NotificationTypeSystem   NotificationType = "system"
)

// Notification represents an in-app notification sent to a user
type Notification struct {
	ID              uuid.UUID        `gorm:"type:char(36);primaryKey" json:"id"`
	UserID          uuid.UUID        `gorm:"type:char(36);index;not null" json:"user_id"`
	Title           string           `gorm:"type:varchar(255);not null" json:"title"`
	Message         string           `gorm:"type:text;not null" json:"message"`
	Type            NotificationType `gorm:"type:varchar(50);not null" json:"type"`
	IsRead          bool             `gorm:"default:false;index" json:"is_read"`
	RelatedEntityID *uuid.UUID       `gorm:"type:char(36);index" json:"related_entity_id"` // E.g., leave request ID
	CreatedAt       time.Time        `gorm:"autoCreateTime" json:"created_at"`
	UpdatedAt       time.Time        `gorm:"autoUpdateTime" json:"updated_at"`

	// Associations - optional, but useful if we need to load the user
	User *User `gorm:"foreignKey:UserID" json:"user,omitempty"`
}

// BeforeCreate will set a UUID rather than numeric ID.
func (n *Notification) BeforeCreate(tx *gorm.DB) error {
	if n.ID == uuid.Nil {
		n.ID = uuid.New()
	}
	return nil
}
