package services

import (
	"errors"
	"fmt"
	"leave-management-system/internal/models"
	"strings"
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
	departmentSvc      *DepartmentService
}

func NewLeaveService(db *gorm.DB, calculator *LeaveCalculator,
	auditLogger *logger.AuditLogger, holidayService *HolidayService, leaveTypeConfigSvc *LeaveTypeConfigService, departmentSvc *DepartmentService) *LeaveService {
	return &LeaveService{
		db:                 db,
		calculator:         calculator,
		auditLogger:        auditLogger,
		holidayService:     holidayService,
		leaveTypeConfigSvc: leaveTypeConfigSvc,
		workflowSvc:        NewWorkflowService(db, departmentSvc),
		departmentSvc:      departmentSvc,
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
		} else if user.DepartmentID != nil && ls.departmentSvc != nil {
			// No direct manager — try department HOD
			approver, err := ls.departmentSvc.ResolveApproverForDepartment(*user.DepartmentID)
			if err == nil && approver != nil {
				request.ApproverID = &approver.ID
			} else {
				// No HOD for department either, escalate to HR
				request.Status = models.StatusEscalated
				request.IsEscalated = true
				now := time.Now()
				request.EscalatedAt = &now
			}
		} else {
			// If no manager and no department, escalate to HR
			request.Status = models.StatusEscalated
			request.IsEscalated = true
			now := time.Now()
			request.EscalatedAt = &now
		}

		// Save request first
		if err := tx.Create(request).Error; err != nil {
			return err
		}

		// Initialize workflow state - MANDATORY
		workflowState, err := ls.workflowSvc.InitializeWorkflowState(request)
		if err != nil {
			return fmt.Errorf("failed to initialize workflow: %w", err)
		}

		// Update request with workflow state ID
		request.WorkflowStateID = &workflowState.ID
		if err := tx.Save(request).Error; err != nil {
			return err
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

		// Get Approver details
		var approver models.User
		if err := tx.First(&approver, "id = ?", approverID).Error; err != nil {
			return err
		}

		// MANDATORY: Workflow State must exist
		var workflowState models.LeaveRequestWorkflowState
		if err := tx.Preload("CurrentStep").Where("leave_request_id = ?", requestID).First(&workflowState).Error; err != nil {
			return fmt.Errorf("active workflow not found for request: %w", err)
		}

		if workflowState.CurrentStep == nil {
			return errors.New("workflow state exists but has no current step")
		}

		// Use WorkflowService to validate responsible users
		responsibleUsers, err := ls.workflowSvc.GetResponsibleUsers(&workflowState)
		if err != nil {
			return fmt.Errorf("failed to determine responsible users: %w", err)
		}

		isAuthorized := false
		for _, u := range responsibleUsers {
			if u.ID == approverID {
				isAuthorized = true
				break
			}
		}

		// Special case: SysAdmin override (optional, but good for safety)
		if !isAuthorized && approver.Role == models.RoleSysAdmin {
			isAuthorized = true
		}

		if !isAuthorized {
			return fmt.Errorf("user %s is not authorized to approve this step (requires: %s)",
				approver.ID, workflowState.CurrentStep.ResponsibleRole)
		}

		// Process Action
		updatedState, err := ls.workflowSvc.ProcessActionWithTx(tx, requestID, models.StepActionApproved, approverID, comment)
		if err != nil {
			return err
		}

		// Update Request Status if Workflow Completed
		if updatedState.IsComplete {
			request.Status = updatedState.FinalStatus
			now := time.Now()
			switch request.Status {
			case models.StatusApproved:
				request.ApprovedAt = &now
			case models.StatusRejected:
				request.RejectedAt = &now
				request.RejectionReason = comment
			}
			request.UpdatedAt = now

			if err := tx.Save(&request).Error; err != nil {
				return err
			}

			// Deduct balance if approved and applicable
			if request.Status == models.StatusApproved {
				if err := ls.deductLeaveBalance(tx, &request); err != nil {
					return err
				}
			}
		}

		return nil
	})
}

// Helper to deduct balance (extracted for clarity)
func (ls *LeaveService) deductLeaveBalance(tx *gorm.DB, request *models.LeaveRequest) error {
	if request.LeaveType == models.LeaveTypeAnnual ||
		request.LeaveType == models.LeaveTypeEmergency ||
		request.LeaveType == models.LeaveTypeSick {

		// Recalculate duration if it's 0
		durationToDeduct := request.DurationDays
		if durationToDeduct <= 0 {
			durationToDeduct = 1
		}

		var balance models.LeaveBalance
		err := tx.Where("user_id = ? AND year = ? AND leave_type = ?",
			request.UserID, request.StartDate.Year(), request.LeaveType).
			First(&balance).Error

		if err != nil {
			// If balance not found, that's an issue since we checked it at creation
			// But maybe we should just create/ignore?
			// Strict consistency: should exist.
			return fmt.Errorf("failed to update balance: %w", err)
		}

		balance.Used += durationToDeduct
		balance.UpdatedAt = time.Now()

		return tx.Save(&balance).Error
	}
	return nil
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
// Handle year-end carry forward
func (ls *LeaveService) ProcessYearEndCarryForward() error {
	return ls.db.Transaction(func(tx *gorm.DB) error {
		// Get all active leave types that allow carry forward
		configs, err := ls.leaveTypeConfigSvc.GetAllConfigs()
		if err != nil {
			return err
		}

		currentYear := time.Now().Year()

		for _, config := range configs {
			if !config.IsActive || !config.AllowCarryForward {
				continue
			}

			// Get all balances for this leave type for current year
			var balances []models.LeaveBalance
			err := tx.Where("year = ? AND leave_type = ?", currentYear, config.LeaveType).
				Find(&balances).Error

			if err != nil {
				return err
			}

			for _, balance := range balances {
				// Calculate unused leave (considering adjustments)
				available := balance.TotalEntitlement + balance.Adjusted - balance.Used
				if available > 0 {
					// Use config max carry forward
					maxCarryForward := float64(config.MaxCarryForwardDays)

					carryForward := available
					if carryForward > maxCarryForward {
						carryForward = maxCarryForward
					}

					// Check if next year's balance already exists
					var nextYearBalance models.LeaveBalance
					err := tx.Where("user_id = ? AND year = ? AND leave_type = ?",
						balance.UserID, currentYear+1, config.LeaveType).First(&nextYearBalance).Error

					if err == nil {
						// Update existing balance
						nextYearBalance.CarriedForward = carryForward
						nextYearBalance.UpdatedAt = time.Now()
						if err := tx.Save(&nextYearBalance).Error; err != nil {
							return err
						}
					} else if errors.Is(err, gorm.ErrRecordNotFound) {
						// Create next year's balance with carried forward amount
						nextYearBalance = models.LeaveBalance{
							ID:               uuid.New(),
							UserID:           balance.UserID,
							LeaveType:        config.LeaveType,
							Year:             currentYear + 1,
							TotalEntitlement: ls.calculateDefaultEntitlementForNextYear(balance.UserID, currentYear+1), // This method might default to Annual, need check
							CarriedForward:   carryForward,
							CreatedAt:        time.Now(),
							UpdatedAt:        time.Now(),
						}

						// Fix: calculateDefaultEntitlementForNextYear currently defaults to Annual.
						// We need to call calculateDefaultEntitlement with specific leave type
						var u models.User
						if err := tx.First(&u, "id = ?", balance.UserID).Error; err == nil {
							nextYearBalance.TotalEntitlement = ls.calculateDefaultEntitlement(&u, currentYear+1, config.LeaveType)
						}

						if err := tx.Create(&nextYearBalance).Error; err != nil {
							return err
						}
					} else {
						return err
					}
				}
			}
		}

		return nil
	})
}

// === New Methods Added by User ===

func (ls *LeaveService) GetUserLeaveRequests(userID uuid.UUID, status, year, leaveType string) ([]models.LeaveRequest, error) {
	var requests []models.LeaveRequest

	query := ls.db.Preload("User").Preload("Approver").Preload("WorkflowState.CurrentStep").Where("user_id = ?", userID)

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
	err := ls.db.Transaction(func(tx *gorm.DB) error {
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

		if err := tx.Create(&chronology).Error; err != nil {
			return err
		}

		// Also cancel the workflow
		if err := ls.workflowSvc.CancelWorkflowWithTx(tx, requestID, userID, "Request cancelled by user"); err != nil {
			return fmt.Errorf("failed to cancel workflow: %w", err)
		}

		return nil
	})

	return err
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

		// Get Approver details
		var approver models.User
		if err := tx.First(&approver, "id = ?", approverID).Error; err != nil {
			return err
		}

		// MANDATORY: Workflow State must exist
		var workflowState models.LeaveRequestWorkflowState
		if err := tx.Preload("CurrentStep").Where("leave_request_id = ?", requestID).First(&workflowState).Error; err != nil {
			return fmt.Errorf("active workflow not found for request: %w", err)
		}

		if workflowState.CurrentStep == nil {
			return errors.New("workflow state exists but has no current step")
		}

		// Use WorkflowService to validate responsible users
		responsibleUsers, err := ls.workflowSvc.GetResponsibleUsers(&workflowState)
		if err != nil {
			return fmt.Errorf("failed to determine responsible users: %w", err)
		}

		isAuthorized := false
		for _, u := range responsibleUsers {
			if u.ID == approverID {
				isAuthorized = true
				break
			}
		}

		// Special case: SysAdmin can override
		if !isAuthorized && approver.Role == models.RoleSysAdmin {
			isAuthorized = true
		}

		if !isAuthorized {
			return fmt.Errorf("user %s is not authorized to reject this step (requires: %s)",
				approver.ID, workflowState.CurrentStep.ResponsibleRole)
		}

		// Process Action
		updatedState, err := ls.workflowSvc.ProcessActionWithTx(tx, requestID, models.StepActionRejected, approverID, comment)
		if err != nil {
			return err
		}

		// Update Request Status if Workflow Completed
		if updatedState.IsComplete {
			request.Status = updatedState.FinalStatus
			now := time.Now()
			if request.Status == models.StatusRejected {
				request.RejectedAt = &now
				request.RejectionReason = comment
			}
			request.UpdatedAt = now

			if err := tx.Save(&request).Error; err != nil {
				return err
			}
		}

		return nil
	})
}

func (ls *LeaveService) GetTeamLeaveRequests(managerID uuid.UUID, status, year string) ([]models.LeaveRequest, error) {
	var requests []models.LeaveRequest

	// Start with requests from direct reports
	query := ls.db.Preload("User").
		Preload("Approver").
		Preload("WorkflowState.CurrentStep").
		Joins("JOIN users ON users.id = leave_requests.user_id").
		Where("users.manager_id = ?", managerID)

	if status != "" {
		query = query.Where("leave_requests.status = ?", status)
	}

	if year != "" {
		query = query.Where("EXTRACT(YEAR FROM leave_requests.start_date) = ?", year)
	}

	if err := query.Order("leave_requests.created_at DESC").Find(&requests).Error; err != nil {
		return nil, err
	}

	// Also include department-wide requests if the user is HOD or acting HOD
	if ls.departmentSvc != nil {
		// Check if user is HOD of any department
		var departments []models.Department
		ls.db.Where("hod_id = ?", managerID).Find(&departments)

		// Check if user is acting HOD of any department
		actingDeptIDs, _ := ls.departmentSvc.IsUserActingHOD(managerID)

		// Collect all department IDs
		var deptIDs []uuid.UUID
		for _, dept := range departments {
			deptIDs = append(deptIDs, dept.ID)
		}
		deptIDs = append(deptIDs, actingDeptIDs...)

		// Fetch department members' requests (avoiding duplicates)
		if len(deptIDs) > 0 {
			var deptRequests []models.LeaveRequest
			deptQuery := ls.db.Preload("User").
				Preload("Approver").
				Preload("WorkflowState.CurrentStep").
				Joins("JOIN users ON users.id = leave_requests.user_id").
				Where("users.department_id IN ?", deptIDs).
				Where("users.id != ?", managerID) // Exclude own requests

			if status != "" {
				deptQuery = deptQuery.Where("leave_requests.status = ?", status)
			}
			if year != "" {
				deptQuery = deptQuery.Where("EXTRACT(YEAR FROM leave_requests.start_date) = ?", year)
			}

			if err := deptQuery.Order("leave_requests.created_at DESC").Find(&deptRequests).Error; err == nil {
				// Merge and deduplicate
				seen := make(map[uuid.UUID]bool)
				for _, r := range requests {
					seen[r.ID] = true
				}
				for _, r := range deptRequests {
					if !seen[r.ID] {
						requests = append(requests, r)
						seen[r.ID] = true
					}
				}
			}
		}
	}

	return requests, nil
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

	query := ls.db.Preload("User").Preload("Approver").Preload("WorkflowState.CurrentStep")

	if status != "" {
		query = query.Where("status = ?", status)
	}

	if year != "" {
		query = query.Where("EXTRACT(YEAR FROM start_date) = ?", year)
	}

	if department != "" {
		query = query.Joins("JOIN users ON users.id = leave_requests.user_id").
			Where("users.department_id = ?", department)
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
	if err := ls.db.Order("last_name, first_name").Find(&users).Error; err != nil {
		return nil, err
	}

	// Define all leave types to include in report
	leaveTypes := []models.LeaveType{
		models.LeaveTypeAnnual,
		models.LeaveTypeSick,
		models.LeaveTypeHospitalization,
		models.LeaveTypeMaternity,
		models.LeaveTypePaternity,
		models.LeaveTypeEmergency,
		models.LeaveTypeUnpaid,
		models.LeaveTypeUnrecorded,
	}

	// Build CSV header dynamically
	csv := "Employee ID,Email,First Name,Last Name,Position"
	for _, lt := range leaveTypes {
		// Capitalize first letter for header (e.g., "Annual Leave")
		header := string(lt)
		if len(header) > 0 {
			header = strings.ToUpper(header[:1]) + header[1:]
		}
		csv += fmt.Sprintf(",%s Leave", header)
	}
	csv += ",Total Leave Days\n"

	for _, user := range users {
		// Get leave requests for the month
		var requests []models.LeaveRequest
		query := ls.db.Where("user_id = ? AND status = ?", user.ID, models.StatusApproved)

		if month != "" && year != "" {
			// Filter by start date falling in the selected month/year
			// Note: This matches leaves STARTING in the month.
			// Ideally payroll might want leaves OVERLAPPING, but start date is a standard simple convention.
			query = query.Where("EXTRACT(MONTH FROM start_date) = ? AND EXTRACT(YEAR FROM start_date) = ? top", month, year)
		}

		if err := query.Find(&requests).Error; err != nil {
			continue
		}

		// Initialize usage map
		usage := make(map[models.LeaveType]float64)
		var totalDays float64

		// Calculate usage per type
		for _, req := range requests {
			usage[req.LeaveType] += req.DurationDays
			totalDays += req.DurationDays
		}

		// Build row
		row := fmt.Sprintf("%s,%s,%s,%s,%s",
			user.ID.String(),
			user.Email,
			user.FirstName,
			user.LastName,
			user.Position,
		)

		// Append usage for each type in correct order
		for _, lt := range leaveTypes {
			days := usage[lt]
			row += fmt.Sprintf(",%.1f", days)
		}

		row += fmt.Sprintf(",%.1f\n", totalDays)
		csv += row
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

// GetDashboardStats aggregates statistics for the dashboard
func (ls *LeaveService) GetDashboardStats() (map[string]interface{}, error) {
	stats := make(map[string]interface{})

	// 1. Counts by Status
	var statusCounts []struct {
		Status string
		Count  int64
	}
	if err := ls.db.Model(&models.LeaveRequest{}).Select("status, count(*) as count").Group("status").Scan(&statusCounts).Error; err != nil {
		return nil, err
	}

	stats["status_counts"] = statusCounts

	// 2. Approved Leaves by Type (for Pie Chart)
	var typeCounts []struct {
		LeaveType models.LeaveType
		Count     int64
	}
	if err := ls.db.Model(&models.LeaveRequest{}).Where("status = ?", models.StatusApproved).Select("leave_type, count(*) as count").Group("leave_type").Scan(&typeCounts).Error; err != nil {
		return nil, err
	}
	stats["type_counts"] = typeCounts

	// 3. Recent Activity (Latest 10 requests)
	var recentRequests []models.LeaveRequest
	if err := ls.db.Preload("User").Order("created_at desc").Limit(10).Find(&recentRequests).Error; err != nil {
		return nil, err
	}
	stats["recent_activity"] = recentRequests

	return stats, nil
}
