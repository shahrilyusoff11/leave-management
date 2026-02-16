package services

import (
	"errors"
	"leave-management-system/internal/models"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type DelegationService struct {
	db *gorm.DB
}

func NewDelegationService(db *gorm.DB) *DelegationService {
	return &DelegationService{db: db}
}

// CreateDelegation creates a new delegation
func (ds *DelegationService) CreateDelegation(delegation *models.UserDelegation) error {
	// 1. Validate dates
	if delegation.EndDate.Before(delegation.StartDate) {
		return errors.New("end date must be after start date")
	}

	// 2. Validate self-delegation
	if delegation.DelegatorID == delegation.DelegateID {
		return errors.New("cannot delegate to yourself")
	}

	// 3. Check for overlapping delegations for the same delegator
	var count int64
	err := ds.db.Model(&models.UserDelegation{}).
		Where("delegator_id = ? AND status = ? AND ((start_date <= ? AND end_date >= ?) OR (start_date <= ? AND end_date >= ?))",
			delegation.DelegatorID, "active",
			delegation.EndDate, delegation.StartDate, // New end overlaps existing start OR
			delegation.StartDate, delegation.EndDate, // New start overlaps existing end
		).Count(&count).Error

	if err != nil {
		return err
	}
	if count > 0 {
		return errors.New("conflicting delegation exists for this period")
	}

	// 4. Create
	return ds.db.Create(delegation).Error
}

// GetActiveDelegation returns the active delegate for a manager on a specific date
// Returns nil if no delegation exists
func (ds *DelegationService) GetActiveDelegation(managerID uuid.UUID, date time.Time) (*models.User, error) {
	var delegation models.UserDelegation
	err := ds.db.Preload("Delegate").
		Where("delegator_id = ? AND status = ? AND start_date <= ? AND end_date >= ?",
			managerID, "active", date, date).
		First(&delegation).Error

	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil // No active delegation
		}
		return nil, err
	}

	return delegation.Delegate, nil
}

// GetDelegations returns all delegations created by a user
func (ds *DelegationService) GetDelegations(userID uuid.UUID) ([]models.UserDelegation, error) {
	var delegations []models.UserDelegation
	err := ds.db.Preload("Delegate").
		Where("delegator_id = ?", userID).
		Order("start_date DESC").
		Find(&delegations).Error
	return delegations, err
}

// CancelDelegation cancels a delegation
func (ds *DelegationService) CancelDelegation(id, userID uuid.UUID) error {
	return ds.db.Model(&models.UserDelegation{}).
		Where("id = ? AND delegator_id = ?", id, userID).
		Update("status", "cancelled").Error
}
