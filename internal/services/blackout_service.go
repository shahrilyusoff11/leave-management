package services

import (
	"errors"
	"leave-management-system/internal/models"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type BlackoutDateService struct {
	db *gorm.DB
}

func NewBlackoutDateService(db *gorm.DB) *BlackoutDateService {
	return &BlackoutDateService{db: db}
}

func (s *BlackoutDateService) CreateBlackoutDate(req *models.BlackoutDate) error {
	if req.StartDate.After(req.EndDate) {
		return errors.New("start date cannot be after end date")
	}
	return s.db.Create(req).Error
}

func (s *BlackoutDateService) UpdateBlackoutDate(id uuid.UUID, req *models.BlackoutDate) error {
	var existing models.BlackoutDate
	if err := s.db.First(&existing, "id = ?", id).Error; err != nil {
		return err
	}

	if req.StartDate.After(req.EndDate) {
		return errors.New("start date cannot be after end date")
	}

	existing.StartDate = req.StartDate
	existing.EndDate = req.EndDate
	existing.Reason = req.Reason
	existing.ApplyToAll = req.ApplyToAll
	existing.DepartmentID = req.DepartmentID
	existing.LeaveType = req.LeaveType

	return s.db.Save(&existing).Error
}

func (s *BlackoutDateService) DeleteBlackoutDate(id uuid.UUID) error {
	return s.db.Delete(&models.BlackoutDate{}, "id = ?", id).Error
}

func (s *BlackoutDateService) GetBlackoutDates() ([]models.BlackoutDate, error) {
	var dates []models.BlackoutDate
	err := s.db.Preload("Department").Preload("Creator").Order("start_date desc").Find(&dates).Error
	return dates, err
}

// CheckIfBlackoutPeriod validates if any date between startDate and endDate is blacked out
// for a specific department and leave type.
func (s *BlackoutDateService) CheckIfBlackoutPeriod(startDate, endDate time.Time, deptID *uuid.UUID, leaveType models.LeaveType) error {
	var blackoutDates []models.BlackoutDate

	// Find any blackout dates that overlap with [startDate, endDate]
	// Overlap condition: StartDate <= requestedEndDate AND EndDate >= requestedStartDate
	err := s.db.Where("start_date <= ? AND end_date >= ?", endDate, startDate).Find(&blackoutDates).Error
	if err != nil {
		return err
	}

	for _, bd := range blackoutDates {
		// Check if it applies
		if bd.ApplyToAll {
			return errors.New("leave period overlaps with a company-wide blackout date: " + bd.Reason)
		}
		if bd.DepartmentID != nil && deptID != nil && *bd.DepartmentID == *deptID {
			return errors.New("leave period overlaps with a department blackout date: " + bd.Reason)
		}
		if bd.LeaveType != nil && *bd.LeaveType == leaveType {
			return errors.New("leave period overlaps with a blackout date for this leave type: " + bd.Reason)
		}
	}

	return nil
}
