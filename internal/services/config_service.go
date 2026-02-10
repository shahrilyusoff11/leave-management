package services

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// SystemConfig represents system-wide configuration stored in database
type SystemConfig struct {
	ID                  uuid.UUID `gorm:"type:uuid;primary_key" json:"id"`
	CompanyName         string    `gorm:"default:'Company Name'" json:"company_name"`
	Timezone            string    `gorm:"default:'UTC'" json:"timezone"`
	DateFormat          string    `gorm:"default:'YYYY-MM-DD'" json:"date_format"`
	IsEmailEnabled      bool      `gorm:"default:true" json:"is_email_enabled"`
	MaxCarryForwardDays int       `gorm:"default:5" json:"max_carry_forward_days"` // Deprecated: Moved to LeaveTypeConfig
	WorkingDays         string    `gorm:"type:text" json:"-"`                      // JSON array stored as string
	EscalationDays      int       `gorm:"default:7" json:"escalation_days"`        // Deprecated: Moved to LeaveTypeConfig or Workflow
	CreatedAt           time.Time `json:"created_at"`
	UpdatedAt           time.Time `json:"updated_at"`
}

// SystemConfigResponse is the API response format
type SystemConfigResponse struct {
	CompanyName         string   `json:"company_name"`
	Timezone            string   `json:"timezone"`
	DateFormat          string   `json:"date_format"`
	IsEmailEnabled      bool     `json:"is_email_enabled"`
	MaxCarryForwardDays int      `json:"max_carry_forward_days"`
	WorkingDays         []string `json:"working_days"`
	EscalationDays      int      `json:"escalation_days"`
}

type SystemConfigRequest struct {
	CompanyName         string   `json:"company_name"`
	Timezone            string   `json:"timezone"`
	DateFormat          string   `json:"date_format"`
	IsEmailEnabled      bool     `json:"is_email_enabled"`
	MaxCarryForwardDays int      `json:"max_carry_forward_days"`
	WorkingDays         []string `json:"working_days"`
	EscalationDays      int      `json:"escalation_days"`
}

type ConfigService struct {
	db *gorm.DB
}

func NewConfigService(db *gorm.DB) *ConfigService {
	return &ConfigService{db: db}
}

func (s *ConfigService) GetSystemConfig() (*SystemConfigResponse, error) {
	var config SystemConfig

	err := s.db.First(&config).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			// Return default config if none exists
			return &SystemConfigResponse{
				CompanyName:         "Company Name",
				Timezone:            "UTC",
				DateFormat:          "YYYY-MM-DD",
				IsEmailEnabled:      true,
				MaxCarryForwardDays: 5,
				WorkingDays:         []string{"Monday", "Tuesday", "Wednesday", "Thursday", "Friday"},
				EscalationDays:      7,
			}, nil
		}
		return nil, err
	}

	// Parse working days JSON
	var workingDays []string
	if config.WorkingDays != "" {
		if err := json.Unmarshal([]byte(config.WorkingDays), &workingDays); err != nil {
			workingDays = []string{"Monday", "Tuesday", "Wednesday", "Thursday", "Friday"}
		}
	} else {
		workingDays = []string{"Monday", "Tuesday", "Wednesday", "Thursday", "Friday"}
	}

	return &SystemConfigResponse{
		CompanyName:         config.CompanyName,
		Timezone:            config.Timezone,
		DateFormat:          config.DateFormat,
		IsEmailEnabled:      config.IsEmailEnabled,
		MaxCarryForwardDays: config.MaxCarryForwardDays,
		WorkingDays:         workingDays,
		EscalationDays:      config.EscalationDays,
	}, nil
}

func (s *ConfigService) UpdateSystemConfig(req SystemConfigRequest) error {
	// Serialize working days to JSON
	workingDaysJSON, err := json.Marshal(req.WorkingDays)
	if err != nil {
		return err
	}

	var config SystemConfig
	err = s.db.First(&config).Error

	if err == gorm.ErrRecordNotFound {
		// Create new config
		config = SystemConfig{
			ID:                  uuid.New(),
			CompanyName:         req.CompanyName,
			Timezone:            req.Timezone,
			DateFormat:          req.DateFormat,
			IsEmailEnabled:      req.IsEmailEnabled,
			MaxCarryForwardDays: req.MaxCarryForwardDays,
			WorkingDays:         string(workingDaysJSON),
			EscalationDays:      req.EscalationDays,
			CreatedAt:           time.Now(),
			UpdatedAt:           time.Now(),
		}
		return s.db.Create(&config).Error
	} else if err != nil {
		return err
	}

	// Update existing config
	config.CompanyName = req.CompanyName
	config.Timezone = req.Timezone
	config.DateFormat = req.DateFormat
	config.IsEmailEnabled = req.IsEmailEnabled
	config.MaxCarryForwardDays = req.MaxCarryForwardDays
	config.WorkingDays = string(workingDaysJSON)
	config.EscalationDays = req.EscalationDays
	config.UpdatedAt = time.Now()

	return s.db.Save(&config).Error
}

func (s *ConfigService) GetMaxCarryForwardDays() int {
	config, err := s.GetSystemConfig()
	if err != nil {
		return 5 // Default
	}
	return config.MaxCarryForwardDays
}

func (s *ConfigService) GetEscalationDays() int {
	config, err := s.GetSystemConfig()
	if err != nil {
		return 7 // Default
	}
	return config.EscalationDays
}
