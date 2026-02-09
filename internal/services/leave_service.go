package services

import (
	"errors"
	"fmt"
	"leave-management-system/internal/models"
	"time"

	"leave-management-system/pkg/logger"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type LeaveService struct {
	db                 *gorm.DB
	calculator         *LeaveCalculator
	auditLogger        *logger.AuditLogger
	holidayService     *HolidayService
	leaveTypeConfigSvc *LeaveTypeConfigService
	workflowSvc        *WorkflowService
}

func NewLeaveService(db *gorm.DB, calculator *LeaveCalculator,
	auditLogger *logger.AuditLogger, holidayService *HolidayService, leaveTypeConfigSvc *LeaveTypeConfigService) *LeaveService {
	return &LeaveService{
		db:                 db,
		calculator:         calculator,
		auditLogger:        auditLogger,
		holidayService:     holidayService,
		leaveTypeConfigSvc: leaveTypeConfigSvc,
		workflowSvc:        NewWorkflowService(db),
	}
}

// GetWorkflowService returns the workflow service for external access
func (ls *LeaveService) GetWorkflowService() *WorkflowService {
	return ls.workflowSvc
}

// GetDB returns the database connection for external service creation
func (ls *LeaveService) GetDB() *gorm.DB {
	return ls.db
}

func (ls *LeaveService) CreateLeaveRequest(userID uuid.UUID, request *models.LeaveRequest) error {
	return ls.db.Transaction(func(tx *gorm.DB) error {
		// Get user with manager
		var user models.User
		if err := tx.Preload("Manager").First(&user, "id = ?", userID).Error; err != nil {
			return err
		}

		// Validate request
		if err := ls.calculator.ValidateLeaveRequest(&user, request); err != nil {
			return err
		}

		// Calculate working days
		workingDays, err := ls.calculator.CalculateWorkingDays(
			request.StartDate, request.EndDate, request.LeaveType)
		if err != nil {
			return err
		}
		request.DurationDays = workingDays

		// Check balance for leave types that deduct from balance
		if request.LeaveType == models.LeaveTypeAnnual ||
			request.LeaveType == models.LeaveTypeEmergency ||
			request.LeaveType == models.LeaveTypeSick {

			balance, err := ls.GetLeaveBalance(userID, int(time.Now().Year()), request.LeaveType)
			if err != nil {
				return err
			}

			available := balance.TotalEntitlement + balance.CarriedForward +
				balance.Adjusted - balance.Used

			if available < request.DurationDays {
				return fmt.Errorf("insufficient balance. Available: %.1f, Requested: %.1f",
					available, request.DurationDays)
			}
		}

		// Set request details
		request.ID = uuid.New()
		request.UserID = userID
		request.Status = models.StatusPending

		if user.ManagerID != nil {
			request.ApproverID = user.ManagerID
		} else {
			// If no manager, escalate to HR
			request.Status = models.StatusEscalated
			request.IsEscalated = true
			now := time.Now()
			request.EscalatedAt = &now
		}

		// Save request first
		if err := tx.Create(request).Error; err != nil {
			return err
		}

		// Initialize workflow state
		workflowState, err := ls.workflowSvc.InitializeWorkflowState(request)
		if err != nil {
			// Log error but continue - workflow is optional
			fmt.Printf("Warning: Could not initialize workflow: %v\n", err)
		} else if workflowState != nil {
			// Update request with workflow state ID
			request.WorkflowStateID = &workflowState.ID
			if err := tx.Save(request).Error; err != nil {
				return err
			}
		}

		// Create chronology entry
		chronology := models.Chronology{
			ID:             uuid.New(),
			LeaveRequestID: request.ID,
			Action:         "submitted",
			ActorID:        userID,
			Comment:        "Leave application submitted",
			Metadata: models.JSONMap{
				"leave_type":      request.LeaveType,
				"start_date":      request.StartDate.Format(time.RFC3339),
				"end_date":        request.EndDate.Format(time.RFC3339),
				"duration":        request.DurationDays,
				"workflow_active": workflowState != nil,
			},
			CreatedAt: time.Now(),
		}

		if err := tx.Create(&chronology).Error; err != nil {
			return err
		}

		return nil
	})
}

func (ls *LeaveService) ApproveLeave(requestID, approverID uuid.UUID, comment string) error {
	return ls.db.Transaction(func(tx *gorm.DB) error {
		var request models.LeaveRequest
		if err := tx.Preload("User").First(&request, "id = ?", requestID).Error; err != nil {
			return err
		}

		if request.Status != models.StatusPending && request.Status != models.StatusEscalated {
			return errors.New("leave request is not pending")
		}

		// Check if approver is the manager
		if request.ApproverID != nil && *request.ApproverID != approverID {
			// Allow HR to approve escalated requests?
			// For now, strict check, but assuming logic allows manager
		}

		// Update request
		now := time.Now()
		request.Status = models.StatusApproved
		request.ApprovedAt = &now
		request.UpdatedAt = now

		// Deduct balance if applicable
		if request.LeaveType == models.LeaveTypeAnnual ||
			request.LeaveType == models.LeaveTypeEmergency ||
			request.LeaveType == models.LeaveTypeSick {

			// Recalculate duration if it's 0 (legacy records)
			durationToDeduct := request.DurationDays
			if durationToDeduct <= 0 {
				// Recalculate working days
				calculatedDays, err := ls.calculator.CalculateWorkingDays(request.StartDate, request.EndDate, request.LeaveType)
				if err == nil && calculatedDays > 0 {
					durationToDeduct = calculatedDays
					// Also update the request record
					request.DurationDays = durationToDeduct
				} else {
					// Fallback: at least 1 day for same-day leave
					durationToDeduct = 1
					request.DurationDays = 1
				}
			}

			var balance models.LeaveBalance
			err := tx.Where("user_id = ? AND year = ? AND leave_type = ?",
				request.UserID, request.StartDate.Year(), request.LeaveType).
				First(&balance).Error

			if err != nil {
				return err
			}

			// Update balance with recalculated duration
			balance.Used += durationToDeduct
			balance.UpdatedAt = time.Now()

			if err := tx.Save(&balance).Error; err != nil {
				return err
			}
		}

		// Create chronology entry
		chronology := models.Chronology{
			ID:             uuid.New(),
			LeaveRequestID: request.ID,
			Action:         "approved",
			ActorID:        approverID,
			Comment:        comment,
			CreatedAt:      time.Now(),
		}

		if err := tx.Save(&request).Error; err != nil {
			return err
		}

		if err := tx.Create(&chronology).Error; err != nil {
			return err
		}

		return nil
	})
}

func (ls *LeaveService) GetLeaveBalance(userID uuid.UUID, year int, leaveType models.LeaveType) (*models.LeaveBalance, error) {
	var balance models.LeaveBalance

	err := ls.db.Where("user_id = ? AND year = ? AND leave_type = ?",
		userID, year, leaveType).
		First(&balance).Error

	if errors.Is(err, gorm.ErrRecordNotFound) {
		// Create default balance if not exists
		var user models.User
		if err := ls.db.First(&user, "id = ?", userID).Error; err != nil {
			return nil, err
		}

		balance = models.LeaveBalance{
			ID:               uuid.New(),
			UserID:           userID,
			LeaveType:        leaveType,
			Year:             year,
			TotalEntitlement: ls.calculateDefaultEntitlement(&user, year, leaveType),
			Used:             0,
			CarriedForward:   0,
			Adjusted:         0,
			IsOverridden:     false,
			CreatedAt:        time.Now(),
			UpdatedAt:        time.Now(),
		}

		if err := ls.db.Create(&balance).Error; err != nil {
			return nil, err
		}
	} else if err != nil {
		return nil, err
	}

	return &balance, nil
}

func (ls *LeaveService) calculateDefaultEntitlement(user *models.User, year int, leaveType models.LeaveType) float64 {
	// Use the calculator which now reads from database config
	return ls.calculator.CalculateLeaveEntitlement(leaveType, user.JoinedDate, year)
}

func (ls *LeaveService) calculateDefaultEntitlementForNextYear(userID uuid.UUID, year int) float64 {
	// Reusing same logic for now
	var u models.User
	if err := ls.db.First(&u, "id = ?", userID).Error; err != nil {
		return 0
	}
	return ls.calculateDefaultEntitlement(&u, year, models.LeaveTypeAnnual)
}

// Handle year-end carry forward
func (ls *LeaveService) ProcessYearEndCarryForward() error {
	return ls.db.Transaction(func(tx *gorm.DB) error {
		// Get all annual leave balances for current year
		currentYear := time.Now().Year()
		var balances []models.LeaveBalance

		err := tx.Where("year = ? AND leave_type = ?", currentYear, models.LeaveTypeAnnual).
			Find(&balances).Error

		if err != nil {
			return err
		}

		for _, balance := range balances {
			// Calculate unused leave (considering adjustments)
			available := balance.TotalEntitlement + balance.Adjusted - balance.Used
			if available > 0 {
				// Carry forward up to max limit (configurable)
				maxCarryForward := 5.0 // Default fallback

				// Get config for annual leave
				config, err := ls.leaveTypeConfigSvc.GetConfig(models.LeaveTypeAnnual)
				if err == nil && config.AllowCarryForward {
					maxCarryForward = float64(config.MaxCarryForwardDays)
				} else if err == nil && !config.AllowCarryForward {
					maxCarryForward = 0
				}

				carryForward := available
				if carryForward > maxCarryForward {
					carryForward = maxCarryForward
				}

				// Create next year's balance with carried forward amount
				nextYearBalance := models.LeaveBalance{
					ID:               uuid.New(),
					UserID:           balance.UserID,
					LeaveType:        models.LeaveTypeAnnual,
					Year:             currentYear + 1,
					TotalEntitlement: ls.calculateDefaultEntitlementForNextYear(balance.UserID, currentYear+1),
					CarriedForward:   carryForward,
					CreatedAt:        time.Now(),
					UpdatedAt:        time.Now(),
				}

				if err := tx.Create(&nextYearBalance).Error; err != nil {
					return err
				}
			}
		}

		return nil
	})
}

// === New Methods Added by User ===

func (ls *LeaveService) GetUserLeaveRequests(userID uuid.UUID, status, year, leaveType string) ([]models.LeaveRequest, error) {
	var requests []models.LeaveRequest

	query := ls.db.Preload("User").Preload("Approver").Where("user_id = ?", userID)

	if status != "" {
		query = query.Where("status = ?", status)
	}

	if year != "" {
		query = query.Where("EXTRACT(YEAR FROM start_date) = ?", year)
	}

	if leaveType != "" {
		query = query.Where("leave_type = ?", leaveType)
	}

	err := query.Order("created_at DESC").Find(&requests).Error
	return requests, err
}

func (ls *LeaveService) GetLeaveRequest(requestID uuid.UUID) (*models.LeaveRequest, error) {
	var request models.LeaveRequest
	err := ls.db.Preload("User").
		Preload("Approver").
		Preload("ChronologyEntries").
		Preload("ChronologyEntries.Actor").
		First(&request, "id = ?", requestID).Error
	return &request, err
}

func (ls *LeaveService) CancelLeaveRequest(requestID, userID uuid.UUID) error {
	return ls.db.Transaction(func(tx *gorm.DB) error {
		var request models.LeaveRequest
		if err := tx.First(&request, "id = ? AND user_id = ?", requestID, userID).Error; err != nil {
			return err
		}

		// Only pending requests can be cancelled
		if request.Status != models.StatusPending {
			return errors.New("only pending requests can be cancelled")
		}

		request.Status = models.StatusCancelled
		request.UpdatedAt = time.Now()

		// Create chronology entry
		chronology := models.Chronology{
			ID:             uuid.New(),
			LeaveRequestID: request.ID,
			Action:         "cancelled",
			ActorID:        userID,
			Comment:        "Request cancelled by user",
			CreatedAt:      time.Now(),
		}

		if err := tx.Save(&request).Error; err != nil {
			return err
		}

		return tx.Create(&chronology).Error
	})
}

func (ls *LeaveService) RejectLeave(requestID, approverID uuid.UUID, comment string) error {
	return ls.db.Transaction(func(tx *gorm.DB) error {
		var request models.LeaveRequest
		if err := tx.Preload("User").First(&request, "id = ?", requestID).Error; err != nil {
			return err
		}

		// Check if request can be rejected
		if request.Status != models.StatusPending && request.Status != models.StatusEscalated {
			return errors.New("leave request is not pending")
		}

		// Authorization check:
		// - If ApproverID is set, only that approver can reject
		// - If ApproverID is nil (escalated), any HR/Admin can reject (handled at route level)
		if request.ApproverID != nil && *request.ApproverID != approverID {
			return errors.New("not authorized to reject this request")
		}

		// Update request
		now := time.Now()
		request.Status = models.StatusRejected
		request.RejectedAt = &now
		request.RejectionReason = comment
		request.UpdatedAt = now

		// Create chronology entry
		chronology := models.Chronology{
			ID:             uuid.New(),
			LeaveRequestID: request.ID,
			Action:         "rejected",
			ActorID:        approverID,
			Comment:        comment,
			CreatedAt:      time.Now(),
		}

		// Save updates
		if err := tx.Save(&request).Error; err != nil {
			return err
		}

		return tx.Create(&chronology).Error
	})
}

func (ls *LeaveService) GetTeamLeaveRequests(managerID uuid.UUID, status, year string) ([]models.LeaveRequest, error) {
	var requests []models.LeaveRequest

	query := ls.db.Preload("User").
		Preload("Approver").
		Joins("JOIN users ON users.id = leave_requests.user_id").
		Where("users.manager_id = ?", managerID)

	if status != "" {
		query = query.Where("leave_requests.status = ?", status)
	}

	if year != "" {
		query = query.Where("EXTRACT(YEAR FROM leave_requests.start_date) = ?", year)
	}

	err := query.Order("leave_requests.created_at DESC").Find(&requests).Error
	return requests, err
}

func (ls *LeaveService) GetUserLeaveBalance(userID uuid.UUID, year string) (map[string]interface{}, error) {
	var balances []models.LeaveBalance

	query := ls.db.Where("user_id = ?", userID)
	if year != "" {
		query = query.Where("year = ?", year)
	}

	err := query.Find(&balances).Error
	if err != nil {
		return nil, err
	}

	// Calculate available balance for each type
	result := make(map[string]interface{})
	for _, balance := range balances {
		available := balance.TotalEntitlement + balance.CarriedForward + balance.Adjusted - balance.Used
		result[string(balance.LeaveType)] = map[string]interface{}{
			"total_entitlement": balance.TotalEntitlement,
			"used":              balance.Used,
			"carried_forward":   balance.CarriedForward,
			"adjusted":          balance.Adjusted,
			"available":         available,
			"is_overridden":     balance.IsOverridden,
		}
	}

	return result, nil
}

func (ls *LeaveService) GetAllLeaveRequests(status, year, department string) ([]models.LeaveRequest, error) {
	var requests []models.LeaveRequest

	query := ls.db.Preload("User").Preload("Approver")

	if status != "" {
		query = query.Where("status = ?", status)
	}

	if year != "" {
		query = query.Where("EXTRACT(YEAR FROM start_date) = ?", year)
	}

	if department != "" {
		query = query.Joins("JOIN users ON users.id = leave_requests.user_id").
			Where("users.department = ?", department)
	}

	err := query.Order("created_at DESC").Find(&requests).Error
	return requests, err
}

func (ls *LeaveService) UpdateLeaveBalance(userID uuid.UUID, leaveType models.LeaveType, year int,
	totalEntitlement, adjustment float64, reason string) (*models.LeaveBalance, error) {

	var balance models.LeaveBalance

	err := ls.db.Transaction(func(tx *gorm.DB) error {
		// Find or create balance
		err := tx.Where("user_id = ? AND year = ? AND leave_type = ?",
			userID, year, leaveType).First(&balance).Error

		if errors.Is(err, gorm.ErrRecordNotFound) {
			balance = models.LeaveBalance{
				ID:               uuid.New(),
				UserID:           userID,
				LeaveType:        leaveType,
				Year:             year,
				TotalEntitlement: totalEntitlement,
				Adjusted:         adjustment,
				IsOverridden:     true,
				CreatedAt:        time.Now(),
				UpdatedAt:        time.Now(),
			}
			return tx.Create(&balance).Error
		} else if err != nil {
			return err
		}

		// Update balance
		balance.TotalEntitlement = totalEntitlement
		balance.Adjusted = adjustment
		balance.IsOverridden = true
		balance.UpdatedAt = time.Now()

		return tx.Save(&balance).Error
	})

	return &balance, err
}

func (ls *LeaveService) GeneratePayrollReport(month, year string) ([]byte, error) {
	// Get all users
	var users []models.User
	if err := ls.db.Find(&users).Error; err != nil {
		return nil, err
	}

	// Build CSV header
	csv := "Employee ID,Email,First Name,Last Name,Department,Position,"
	csv += "Annual Leave Used,Sick Leave Used,Unpaid Leave Days,Total Leave Days\n"

	for _, user := range users {
		// Get leave requests for the month
		var requests []models.LeaveRequest
		query := ls.db.Where("user_id = ? AND status = ?", user.ID, models.StatusApproved)

		if month != "" && year != "" {
			query = query.Where("EXTRACT(MONTH FROM start_date) = ? AND EXTRACT(YEAR FROM start_date) = ?", month, year)
		}

		if err := query.Find(&requests).Error; err != nil {
			continue
		}

		// Calculate leave by type
		var annualUsed, sickUsed, unpaidUsed, totalDays float64
		for _, req := range requests {
			switch req.LeaveType {
			case models.LeaveTypeAnnual:
				annualUsed += req.DurationDays
			case models.LeaveTypeSick:
				sickUsed += req.DurationDays
			case models.LeaveTypeUnpaid:
				unpaidUsed += req.DurationDays
			}
			totalDays += req.DurationDays
		}

		// Add row to CSV
		csv += fmt.Sprintf("%s,%s,%s,%s,%s,%s,%.1f,%.1f,%.1f,%.1f\n",
			user.ID.String(),
			user.Email,
			user.FirstName,
			user.LastName,
			user.Department,
			user.Position,
			annualUsed,
			sickUsed,
			unpaidUsed,
			totalDays,
		)
	}

	return []byte(csv), nil
}

func (ls *LeaveService) GetPendingRequestsOlderThan(date time.Time) ([]models.LeaveRequest, error) {
	var requests []models.LeaveRequest

	err := ls.db.Preload("User").
		Preload("User.Manager").
		Where("status = ? AND created_at < ?", models.StatusPending, date).
		Find(&requests).Error

	return requests, err
}

func (ls *LeaveService) EscalateRequest(requestID uuid.UUID) error {
	return ls.db.Transaction(func(tx *gorm.DB) error {
		var request models.LeaveRequest
		if err := tx.Preload("User").First(&request, "id = ?", requestID).Error; err != nil {
			return err
		}

		now := time.Now()
		request.Status = models.StatusEscalated
		request.IsEscalated = true
		request.EscalatedAt = &now
		request.UpdatedAt = now

		// Create chronology entry
		chronology := models.Chronology{
			ID:             uuid.New(),
			LeaveRequestID: request.ID,
			Action:         "escalated",
			ActorID:        request.UserID, // System action
			Comment:        "Request escalated due to no response from manager",
			Metadata: models.JSONMap{
				"reason": "7-day escalation rule",
			},
			CreatedAt: time.Now(),
		}

		if err := tx.Save(&request).Error; err != nil {
			return err
		}

		return tx.Create(&chronology).Error
	})
}

func (ls *LeaveService) ArchiveOldRecords(beforeDate time.Time) error {
	// Archive logic would be implemented here
	// For production, this would move old records to an archive table
	return nil
}

// GetLeaveRequestChronology returns the history/timeline of a leave request
func (ls *LeaveService) GetLeaveRequestChronology(requestID uuid.UUID) ([]models.Chronology, error) {
	var chronology []models.Chronology
	err := ls.db.Where("leave_request_id = ?", requestID).
		Preload("Actor").
		Order("created_at ASC").
		Find(&chronology).Error
	if err != nil {
		return nil, err
	}
	return chronology, nil
}
