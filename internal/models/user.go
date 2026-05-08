package models

import (
	"fmt"
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

type Role struct {
	ID        uuid.UUID `gorm:"type:uuid;primary_key" json:"id"`
	Name      UserRole  `gorm:"type:varchar(20);uniqueIndex;not null" json:"name"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

type UserRoleAssignment struct {
	UserID    uuid.UUID `gorm:"type:uuid;primaryKey;column:user_id" json:"user_id"`
	RoleID    uuid.UUID `gorm:"type:uuid;primaryKey;column:role_id" json:"role_id"`
	CreatedAt time.Time `json:"created_at"`
}

type User struct {
	ID                uuid.UUID      `gorm:"type:uuid;primary_key" json:"id"`
	Email             string         `gorm:"uniqueIndex;not null" json:"email"`
	PasswordHash      string         `gorm:"not null" json:"-"`
	FirstName         string         `gorm:"not null" json:"first_name"`
	LastName          string         `gorm:"not null" json:"last_name"`
	Role              UserRole       `gorm:"-" json:"role"`
	Roles             []Role         `gorm:"many2many:user_roles;joinForeignKey:UserID;joinReferences:RoleID" json:"roles,omitempty"`
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

func (r *Role) BeforeCreate(tx *gorm.DB) error {
	if r.ID == uuid.Nil {
		r.ID = uuid.New()
	}
	return nil
}

func (UserRoleAssignment) TableName() string {
	return "user_roles"
}

func (u *User) BeforeSave(tx *gorm.DB) error {
	if u.Role == "" && len(u.Roles) > 0 {
		u.Role = PrimaryRoleFromRoles(u.Roles)
	}

	if u.Role == "" && len(u.Roles) == 0 {
		return fmt.Errorf("role is required")
	}

	if len(u.Roles) == 0 {
		role, err := FindOrCreateRole(tx, u.Role)
		if err != nil {
			return err
		}
		u.Roles = []Role{*role}
	}

	return nil
}

func (u *User) AfterFind(tx *gorm.DB) error {
	if len(u.Roles) == 0 {
		if err := tx.Model(u).Association("Roles").Find(&u.Roles); err != nil {
			return err
		}
	}

	if len(u.Roles) > 0 {
		u.Role = PrimaryRoleFromRoles(u.Roles)
		return nil
	}

	return nil
}

func FindOrCreateRole(tx *gorm.DB, roleName UserRole) (*Role, error) {
	var role Role
	if err := tx.Where("name = ?", roleName).First(&role).Error; err != nil {
		if err != gorm.ErrRecordNotFound {
			return nil, err
		}

		role = Role{Name: roleName}
		if err := tx.Create(&role).Error; err != nil {
			return nil, err
		}
	}

	return &role, nil
}

func PrimaryRoleFromRoles(roles []Role) UserRole {
	var primary UserRole
	highest := -1
	for _, role := range roles {
		if level := RoleLevel(role.Name); level > highest {
			highest = level
			primary = role.Name
		}
	}
	return primary
}

func RoleLevel(role UserRole) int {
	switch role {
	case RoleSysAdmin:
		return 100
	case RoleAdmin:
		return 80
	case RoleHR:
		return 60
	case RoleHOD:
		return 40
	case RoleManager:
		return 30
	case RoleStaff:
		return 10
	default:
		return 0
	}
}

func (u *User) HasRole(role UserRole) bool {
	for _, assigned := range u.Roles {
		if assigned.Name == role {
			return true
		}
	}
	return u.Role == role
}

func (u *User) HasAnyRole(roles ...UserRole) bool {
	for _, role := range roles {
		if u.HasRole(role) {
			return true
		}
	}
	return false
}
