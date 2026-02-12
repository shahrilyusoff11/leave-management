package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type UserRole string

const (
	RoleSysAdmin UserRole = "sysadmin"
	RoleAdmin    UserRole = "admin"
	RoleHR       UserRole = "hr"
	RoleManager  UserRole = "manager"
	RoleHOD      UserRole = "hod"
	RoleStaff    UserRole = "staff"
)

type User struct {
	ID                uuid.UUID      `gorm:"type:uuid;primary_key" json:"id"`
	Email             string         `gorm:"uniqueIndex;not null" json:"email"`
	PasswordHash      string         `gorm:"not null" json:"-"`
	FirstName         string         `gorm:"not null" json:"first_name"`
	LastName          string         `gorm:"not null" json:"last_name"`
	Role              UserRole       `gorm:"type:varchar(20);not null" json:"role"`
	DepartmentID      *uuid.UUID     `json:"department_id"`
	DepartmentRef     *Department    `gorm:"foreignKey:DepartmentID" json:"department_ref,omitempty"`
	Position          string         `json:"position"`
	ManagerID         *uuid.UUID     `json:"manager_id"`
	Manager           *User          `gorm:"foreignKey:ManagerID" json:"manager,omitempty"`
	JoinedDate        time.Time      `gorm:"not null" json:"joined_date"`
	ProbationEndDate  *time.Time     `json:"probation_end_date"`
	IsConfirmed       bool           `gorm:"default:false" json:"is_confirmed"`
	IsActive          bool           `gorm:"default:true" json:"is_active"`
	LeaveEntitlements []LeaveBalance `gorm:"foreignKey:UserID" json:"leave_entitlements,omitempty"`
	LeaveRequests     []LeaveRequest `gorm:"foreignKey:UserID" json:"leave_requests,omitempty"`
	ManagedUsers      []User         `gorm:"foreignKey:ManagerID" json:"managed_users,omitempty"`
	CreatedAt         time.Time      `json:"created_at"`
	UpdatedAt         time.Time      `json:"updated_at"`
	LastLoginAt       *time.Time     `json:"last_login_at"`
}

func (u *User) BeforeCreate(tx *gorm.DB) error {
	if u.ID == uuid.Nil {
		u.ID = uuid.New()
	}
	return nil
}
