package services

import (
	"errors"
	"leave-management-system/internal/models"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// DepartmentService handles department and HOD delegation operations
type DepartmentService struct {
	db *gorm.DB
}

func NewDepartmentService(db *gorm.DB) *DepartmentService {
	return &DepartmentService{db: db}
}

// GetAllDepartments returns all departments with HOD preloaded
func (ds *DepartmentService) GetAllDepartments() ([]models.Department, error) {
	var departments []models.Department
	err := ds.db.Preload("HOD").Find(&departments).Error
	return departments, err
}

// GetDepartment returns a single department with HOD and active delegations
func (ds *DepartmentService) GetDepartment(id uuid.UUID) (*models.Department, error) {
	var dept models.Department
	err := ds.db.Preload("HOD").First(&dept, "id = ?", id).Error
	return &dept, err
}

// CreateDepartment creates a new department
func (ds *DepartmentService) CreateDepartment(dept *models.Department) error {
	return ds.db.Create(dept).Error
}

// UpdateDepartment updates a department's name and/or HOD
func (ds *DepartmentService) UpdateDepartment(dept *models.Department) error {
	dept.UpdatedAt = time.Now()
	return ds.db.Save(dept).Error
}

// DeleteDepartment removes a department
func (ds *DepartmentService) DeleteDepartment(id uuid.UUID) error {
	// Check if any users are assigned to the department
	var count int64
	ds.db.Model(&models.User{}).Where("department_id = ?", id).Count(&count)
	if count > 0 {
		return errors.New("cannot delete department with assigned users")
	}

	return ds.db.Delete(&models.Department{}, "id = ?", id).Error
}

// CreateHODDelegation creates a new Acting HOD delegation
func (ds *DepartmentService) CreateHODDelegation(delegation *models.HODDelegation) error {
	// Validate dates
	if delegation.EndDate.Before(delegation.StartDate) {
		return errors.New("end date must be after start date")
	}

	// Validate delegate is not the same as delegator
	if delegation.DelegatorID == delegation.DelegateID {
		return errors.New("delegator and delegate cannot be the same person")
	}

	return ds.db.Create(delegation).Error
}

// GetDelegationsForDepartment returns all delegations for a department
func (ds *DepartmentService) GetDelegationsForDepartment(departmentID uuid.UUID) ([]models.HODDelegation, error) {
	var delegations []models.HODDelegation
	err := ds.db.Preload("Delegator").Preload("Delegate").
		Where("department_id = ?", departmentID).
		Order("start_date DESC").
		Find(&delegations).Error
	return delegations, err
}

// DeleteHODDelegation removes a delegation
func (ds *DepartmentService) DeleteHODDelegation(id uuid.UUID) error {
	return ds.db.Delete(&models.HODDelegation{}, "id = ?", id).Error
}

// ResolveApproverForDepartment returns the acting HOD if there is an active
// delegation for today, otherwise returns the department HOD.
// Returns the user ID of the person who should handle approvals.
func (ds *DepartmentService) ResolveApproverForDepartment(departmentID uuid.UUID) (*models.User, error) {
	// First, get the department with HOD
	var dept models.Department
	if err := ds.db.Preload("HOD").First(&dept, "id = ?", departmentID).Error; err != nil {
		return nil, err
	}

	if dept.HODID == nil {
		return nil, errors.New("department has no HOD assigned")
	}

	// Check for active delegation for today
	now := time.Now()
	var delegation models.HODDelegation
	err := ds.db.Preload("Delegate").
		Where("department_id = ? AND is_active = ? AND start_date <= ? AND end_date >= ?",
			departmentID, true, now, now).
		First(&delegation).Error

	if err == nil {
		// Active delegation found, return the delegate (acting HOD)
		return &delegation.Delegate, nil
	}

	if !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, err
	}

	// No active delegation, return the HOD
	return dept.HOD, nil
}

// GetDepartmentByUserID returns the department for a given user
func (ds *DepartmentService) GetDepartmentByUserID(userID uuid.UUID) (*models.Department, error) {
	var user models.User
	if err := ds.db.First(&user, "id = ?", userID).Error; err != nil {
		return nil, err
	}

	if user.DepartmentID == nil {
		return nil, errors.New("user is not assigned to any department")
	}

	var dept models.Department
	if err := ds.db.Preload("HOD").First(&dept, "id = ?", *user.DepartmentID).Error; err != nil {
		return nil, err
	}

	return &dept, nil
}

// IsUserActingHOD checks if a user is currently acting as HOD for any department
func (ds *DepartmentService) IsUserActingHOD(userID uuid.UUID) ([]uuid.UUID, error) {
	now := time.Now()
	var delegations []models.HODDelegation
	err := ds.db.Where("delegate_id = ? AND is_active = ? AND start_date <= ? AND end_date >= ?",
		userID, true, now, now).
		Find(&delegations).Error
	if err != nil {
		return nil, err
	}

	var deptIDs []uuid.UUID
	for _, d := range delegations {
		deptIDs = append(deptIDs, d.DepartmentID)
	}
	return deptIDs, nil
}
