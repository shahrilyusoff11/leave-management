package services

import (
	"fmt"
	"leave-management-system/internal/models"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// LeaveTypeConfigService handles leave type configuration operations
type LeaveTypeConfigService struct {
	db *gorm.DB
}

func NewLeaveTypeConfigService(db *gorm.DB) *LeaveTypeConfigService {
	return &LeaveTypeConfigService{db: db}
}

// GetAllConfigs returns all leave type configurations
func (s *LeaveTypeConfigService) GetAllConfigs() ([]models.LeaveTypeConfig, error) {
	var configs []models.LeaveTypeConfig
	err := s.db.Order("display_order ASC").Find(&configs).Error
	return configs, err
}

// GetConfig returns configuration for a specific leave type
func (s *LeaveTypeConfigService) GetConfig(leaveType models.LeaveType) (*models.LeaveTypeConfig, error) {
	var config models.LeaveTypeConfig
	err := s.db.Where("leave_type = ?", leaveType).First(&config).Error
	if err != nil {
		return nil, err
	}
	return &config, nil
}

// UpdateConfig updates configuration for a specific leave type
func (s *LeaveTypeConfigService) UpdateConfig(leaveType models.LeaveType, updates map[string]interface{}) error {
	var config models.LeaveTypeConfig
	if err := s.db.Where("leave_type = ?", leaveType).First(&config).Error; err != nil {
		return err
	}

	// Update fields
	if v, ok := updates["base_entitlement"]; ok {
		if f, ok := v.(float64); ok {
			config.BaseEntitlement = f
		}
	}
	if v, ok := updates["prorate_type"]; ok {
		if s, ok := v.(string); ok {
			config.ProrateType = s
		}
	}
	if v, ok := updates["allow_carry_forward"]; ok {
		if b, ok := v.(bool); ok {
			config.AllowCarryForward = b
		}
	}
	if v, ok := updates["max_carry_forward_days"]; ok {
		if f, ok := v.(float64); ok {
			config.MaxCarryForwardDays = int(f)
		}
	}
	if v, ok := updates["requires_attachment"]; ok {
		if b, ok := v.(bool); ok {
			config.RequiresAttachment = b
		}
	}
	if v, ok := updates["min_advance_days"]; ok {
		if f, ok := v.(float64); ok {
			config.MinAdvanceDays = int(f)
		}
	}
	if v, ok := updates["is_active"]; ok {
		if b, ok := v.(bool); ok {
			config.IsActive = b
		}
	}

	// Handle JSONB field
	if v, ok := updates["years_of_service_tiers"]; ok {
		if tiers, ok := v.(map[string]interface{}); ok {
			config.YearsOfServiceTiers = models.JSONMap(tiers)
		} else if tiers, ok := v.(models.JSONMap); ok {
			config.YearsOfServiceTiers = tiers
		} else if v == nil {
			config.YearsOfServiceTiers = nil
		}
	}

	config.UpdatedAt = time.Now()

	return s.db.Save(&config).Error
}

// SeedDefaultConfigs creates default configurations if none exist
func (s *LeaveTypeConfigService) SeedDefaultConfigs() error {
	// Migration: Rename 'special' to 'unrecorded'
	// This ensures existing databases are updated to the new terminology
	if err := s.db.Model(&models.LeaveTypeConfig{}).
		Where("leave_type = ?", "special").
		Update("leave_type", models.LeaveTypeUnrecorded).Error; err != nil {
		return err // Log warning or return error? Error is safer.
	}

	var count int64
	s.db.Model(&models.LeaveTypeConfig{}).Count(&count)
	if count > 0 {
		return nil // Already seeded
	}

	defaultConfigs := []models.LeaveTypeConfig{
		{
			ID:                  uuid.New(),
			LeaveType:           models.LeaveTypeAnnual,
			BaseEntitlement:     12,
			YearsOfServiceTiers: models.JSONMap{"2": 4, "5": 8},
			ProrateType:         "continuous",
			AllowCarryForward:   true,
			MaxCarryForwardDays: 5,
			IsActive:            true,
			DisplayOrder:        1,
			CreatedAt:           time.Now(),
			UpdatedAt:           time.Now(),
		},
		{
			ID:                  uuid.New(),
			LeaveType:           models.LeaveTypeSick,
			BaseEntitlement:     14,
			YearsOfServiceTiers: models.JSONMap{"2": 4, "5": 8},
			ProrateType:         "none",
			AllowCarryForward:   false,
			RequiresAttachment:  true,
			IsActive:            true,
			DisplayOrder:        2,
			CreatedAt:           time.Now(),
			UpdatedAt:           time.Now(),
		},
		{
			ID:                 uuid.New(),
			LeaveType:          models.LeaveTypeMaternity,
			BaseEntitlement:    98,
			ProrateType:        "none",
			AllowCarryForward:  false,
			RequiresAttachment: true,
			IsActive:           true,
			DisplayOrder:       3,
			CreatedAt:          time.Now(),
			UpdatedAt:          time.Now(),
		},
		{
			ID:                uuid.New(),
			LeaveType:         models.LeaveTypePaternity,
			BaseEntitlement:   7,
			ProrateType:       "none",
			AllowCarryForward: false,
			IsActive:          true,
			DisplayOrder:      4,
			CreatedAt:         time.Now(),
			UpdatedAt:         time.Now(),
		},
		{
			ID:                uuid.New(),
			LeaveType:         models.LeaveTypeEmergency,
			BaseEntitlement:   3,
			ProrateType:       "none",
			AllowCarryForward: false,
			IsActive:          true,
			DisplayOrder:      5,
			CreatedAt:         time.Now(),
			UpdatedAt:         time.Now(),
		},
		{
			ID:                uuid.New(),
			LeaveType:         models.LeaveTypeUnpaid,
			BaseEntitlement:   0,
			ProrateType:       "none",
			AllowCarryForward: false,
			IsActive:          true,
			DisplayOrder:      6,
			CreatedAt:         time.Now(),
			UpdatedAt:         time.Now(),
		},
		{
			ID:                uuid.New(),
			LeaveType:         models.LeaveTypeUnrecorded,
			BaseEntitlement:   0,
			ProrateType:       "none",
			AllowCarryForward: false,
			IsActive:          true,
			DisplayOrder:      7,
			CreatedAt:         time.Now(),
			UpdatedAt:         time.Now(),
		},
		{
			ID:                 uuid.New(),
			LeaveType:          models.LeaveTypeHospitalization,
			BaseEntitlement:    60,
			ProrateType:        "none",
			AllowCarryForward:  false,
			RequiresAttachment: true,
			IsActive:           true,
			DisplayOrder:       8,
			CreatedAt:          time.Now(),
			UpdatedAt:          time.Now(),
		},
	}

	for _, config := range defaultConfigs {
		if err := s.db.Create(&config).Error; err != nil {
			return err
		}
	}

	return nil
}

// GetEntitlement calculates entitlement for a user based on config and years of service
func (s *LeaveTypeConfigService) GetEntitlement(leaveType models.LeaveType, yearsOfService int) float64 {
	config, err := s.GetConfig(leaveType)
	if err != nil {
		// Return hardcoded defaults if config not found
		return s.GetDefaultEntitlement(leaveType, yearsOfService)
	}

	entitlement := config.BaseEntitlement

	// Add years of service bonus
	if config.YearsOfServiceTiers != nil {
		for years, bonus := range config.YearsOfServiceTiers {
			// years is already a string from map[string]interface{}
			var yearsThreshold int
			if _, err := fmt.Sscanf(years, "%d", &yearsThreshold); err != nil {
				continue
			}

			if yearsOfService >= yearsThreshold {
				switch b := bonus.(type) {
				case float64:
					entitlement += b
				case int:
					entitlement += float64(b)
				}
			}
		}
	}

	return entitlement
}

// GetDefaultEntitlement returns hardcoded defaults (fallback)
func (s *LeaveTypeConfigService) GetDefaultEntitlement(leaveType models.LeaveType, yearsOfService int) float64 {
	switch leaveType {
	case models.LeaveTypeAnnual:
		switch {
		case yearsOfService < 2:
			return 12
		case yearsOfService < 5:
			return 12
		default:
			return 16
		}
	case models.LeaveTypeSick:
		switch {
		case yearsOfService < 2:
			return 14
		case yearsOfService < 5:
			return 18
		default:
			return 22
		}
	case models.LeaveTypeMaternity:
		return 98
	case models.LeaveTypePaternity:
		return 7
	case models.LeaveTypeHospitalization:
		return 60
	default:
		return 0
	}
}
