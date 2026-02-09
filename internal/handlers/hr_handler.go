package handlers

import (
	"encoding/json"
	"leave-management-system/internal/models"
	"leave-management-system/internal/services"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
)

type HRHandler struct {
	userService  *services.UserService
	leaveService *services.LeaveService
	auditService *services.AuditService
}

func NewHRHandler(userService *services.UserService, leaveService *services.LeaveService, auditService *services.AuditService) *HRHandler {
	return &HRHandler{
		userService:  userService,
		leaveService: leaveService,
		auditService: auditService,
	}
}

func (h *HRHandler) GetAllUsers(c *gin.Context) {
	users, err := h.userService.GetAllUsers()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, users)
}

func (h *HRHandler) GetUser(c *gin.Context) {
	userID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid user ID"})
		return
	}

	user, err := h.userService.GetUserWithDetails(userID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "User not found"})
		return
	}

	c.JSON(http.StatusOK, user)
}

type CreateUserRequest struct {
	Email           string          `json:"email" binding:"required,email"`
	Password        string          `json:"password" binding:"required,min=8"`
	FirstName       string          `json:"first_name" binding:"required"`
	LastName        string          `json:"last_name" binding:"required"`
	Role            models.UserRole `json:"role" binding:"required"`
	Department      string          `json:"department" binding:"required"`
	Position        string          `json:"position" binding:"required"`
	ManagerID       *uuid.UUID      `json:"manager_id"`
	JoinedDate      string          `json:"joined_date" binding:"required"`
	ProbationMonths int             `json:"probation_months" default:"3"`
}

func (h *HRHandler) CreateUser(c *gin.Context) {
	var req CreateUserRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Validate role permissions
	userRole := c.MustGet("user_role").(models.UserRole)
	if userRole != models.RoleSysAdmin && userRole != models.RoleAdmin &&
		userRole != models.RoleHR {
		c.JSON(http.StatusForbidden, gin.H{"error": "Insufficient permissions to create user"})
		return
	}

	// HR cannot create SysAdmin or Admin users
	if (userRole == models.RoleHR) &&
		(req.Role == models.RoleSysAdmin || req.Role == models.RoleAdmin) {
		c.JSON(http.StatusForbidden, gin.H{"error": "Cannot create admin users"})
		return
	}

	// Hash password
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to hash password"})
		return
	}

	newUser := models.User{
		ID:           uuid.New(),
		Email:        req.Email,
		PasswordHash: string(hashedPassword),
		FirstName:    req.FirstName,
		LastName:     req.LastName,
		Role:         req.Role,
		Department:   req.Department,
		Position:     req.Position,
		ManagerID:    req.ManagerID,
		IsActive:     true,
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	}

	joinedDate, err := time.Parse("2006-01-02", req.JoinedDate)
	if err == nil {
		newUser.JoinedDate = joinedDate
	}

	if err := h.userService.CreateUser(&newUser); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Audit Log (Fire and Forget)
	go func(newUser models.User, ctx *gin.Context) {
		uidVal, _ := ctx.Get("user_id")
		emailVal, _ := ctx.Get("user_email")
		roleVal, _ := ctx.Get("user_role")

		uid, _ := uidVal.(uuid.UUID)
		email, _ := emailVal.(string)
		role, _ := roleVal.(models.UserRole)

		// Helper to convert struct to map
		toMap := func(v interface{}) models.JSONMap {
			b, _ := json.Marshal(v)
			var m models.JSONMap
			json.Unmarshal(b, &m)
			return m
		}

		auditLog := &models.AuditLog{
			ID:          uuid.New(),
			ActorID:     uid,
			ActorEmail:  email,
			ActorRole:   role,
			Action:      "create_user",
			TargetID:    newUser.ID,
			TargetType:  "user",
			BeforeState: nil, // New creation
			AfterState:  toMap(newUser),
			Method:      "POST",
			Endpoint:    ctx.Request.URL.Path,
			IPAddress:   ctx.ClientIP(),
			UserAgent:   ctx.Request.UserAgent(),
			CreatedAt:   time.Now(),
		}
		h.auditService.CreateAuditLog(auditLog)
	}(newUser, c.Copy())

	// Prevent duplicate logging in middleware
	c.Set("audit_logged", true)

	c.JSON(http.StatusCreated, newUser)
}

type UpdateLeaveBalanceRequest struct {
	LeaveType        models.LeaveType `json:"leave_type" binding:"required"`
	Year             int              `json:"year" binding:"required"`
	TotalEntitlement float64          `json:"total_entitlement"`
	Adjustment       float64          `json:"adjustment"`
	Reason           string           `json:"reason" binding:"required"`
}

func (h *HRHandler) UpdateLeaveBalance(c *gin.Context) {
	userID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid user ID"})
		return
	}

	var req UpdateLeaveBalanceRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// 1. Get Before State (We need to fetch the balance first)
	// Note: The service UpdateLeaveBalance method does the update directly.
	// To log "Before" state, we should ideally fetch it first.
	// However, `UpdateLeaveBalance` service might not expose a simple "Get" compatible with the update logic easily without duplicating logic.
	// Let's rely on the `balance` returned which is the "After" state, and we can infer "Before" if needed,
	// OR better: construct a meaningful log with the *Request Parameters* which explain the change.

	// Actually, for financial/balance logs, the "Action" (Adjustment) is often more important than the state snapshot.
	// We will log the *Intent* (Request) and the *Result* (New Balance).

	balance, err := h.leaveService.UpdateLeaveBalance(userID, req.LeaveType, req.Year,
		req.TotalEntitlement, req.Adjustment, req.Reason)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Audit Log (Fire and Forget)
	go func(req UpdateLeaveBalanceRequest, result *models.LeaveBalance, targetUserID uuid.UUID, ctx *gin.Context) {
		uidVal, _ := ctx.Get("user_id")
		emailVal, _ := ctx.Get("user_email")
		roleVal, _ := ctx.Get("user_role")

		uid, _ := uidVal.(uuid.UUID)
		email, _ := emailVal.(string)
		role, _ := roleVal.(models.UserRole)

		// Helper to convert struct to map
		toMap := func(v interface{}) models.JSONMap {
			b, _ := json.Marshal(v)
			var m models.JSONMap
			json.Unmarshal(b, &m)
			return m
		}

		// Construct "Before" state representation (The Request)
		changeDetails := map[string]interface{}{
			"adjustment":        req.Adjustment,
			"total_entitlement": req.TotalEntitlement,
			"reason":            req.Reason,
			"year":              req.Year,
			"leave_type":        req.LeaveType,
		}

		auditLog := &models.AuditLog{
			ID:          uuid.New(),
			ActorID:     uid,
			ActorEmail:  email,
			ActorRole:   role,
			Action:      "update_leave_balance",
			TargetID:    targetUserID,
			TargetType:  "user_leave_balance",
			BeforeState: toMap(changeDetails), // Tracking the *Input/Change* here for clarity
			AfterState:  toMap(result),        // Tracking the *Resulting Balance*
			Method:      "PUT",
			Endpoint:    ctx.Request.URL.Path,
			IPAddress:   ctx.ClientIP(),
			UserAgent:   ctx.Request.UserAgent(),
			CreatedAt:   time.Now(),
		}
		h.auditService.CreateAuditLog(auditLog)
	}(req, balance, userID, c.Copy())

	// Prevent duplicate logging in middleware
	c.Set("audit_logged", true)

	c.JSON(http.StatusOK, balance)
}

func (h *HRHandler) GetLeaveRequests(c *gin.Context) {
	status := c.Query("status")
	year := c.Query("year")
	department := c.Query("department")

	requests, err := h.leaveService.GetAllLeaveRequests(status, year, department)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, requests)
}

func (h *HRHandler) ExportPayrollReport(c *gin.Context) {
	month := c.Query("month")
	year := c.Query("year")

	if month == "" || year == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Month and year are required"})
		return
	}

	report, err := h.leaveService.GeneratePayrollReport(month, year)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.Header("Content-Type", "text/csv")
	c.Header("Content-Disposition", "attachment; filename=payroll_report.csv")
	c.Data(http.StatusOK, "text/csv", report)
}

type ConfirmProbationRequest struct {
	IsConfirmed bool   `json:"is_confirmed"`
	Notes       string `json:"notes"`
}

func (h *HRHandler) ConfirmProbation(c *gin.Context) {
	userID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid user ID"})
		return
	}

	var req ConfirmProbationRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if err := h.userService.UpdateProbationStatus(userID, req.IsConfirmed, req.Notes); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Probation status updated"})
}

type UpdateUserRequest struct {
	FirstName  string          `json:"first_name"`
	LastName   string          `json:"last_name"`
	Role       models.UserRole `json:"role"`
	Department string          `json:"department"`
	Position   string          `json:"position"`
	ManagerID  *uuid.UUID      `json:"manager_id"`
}

func (h *HRHandler) UpdateUser(c *gin.Context) {
	userID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid user ID"})
		return
	}

	var req UpdateUserRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Get existing user
	user, err := h.userService.GetUser(userID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "User not found"})
		return
	}

	// Validate role permissions
	currentRole := c.MustGet("user_role").(models.UserRole)
	if currentRole == models.RoleHR &&
		(req.Role == models.RoleSysAdmin || req.Role == models.RoleAdmin) {
		c.JSON(http.StatusForbidden, gin.H{"error": "Cannot assign admin roles"})
		return
	}

	// Update fields
	if req.FirstName != "" {
		user.FirstName = req.FirstName
	}
	if req.LastName != "" {
		user.LastName = req.LastName
	}
	if req.Role != "" {
		user.Role = req.Role
	}
	if req.Department != "" {
		user.Department = req.Department
	}
	if req.Position != "" {
		user.Position = req.Position
	}
	if req.ManagerID != nil {
		user.ManagerID = req.ManagerID
	}

	if err := h.userService.UpdateUser(user); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, user)
}

type ToggleUserActiveRequest struct {
	IsActive bool `json:"is_active"`
}

func (h *HRHandler) ToggleUserActive(c *gin.Context) {
	userID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid user ID"})
		return
	}

	var req ToggleUserActiveRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Get existing user
	user, err := h.userService.GetUser(userID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "User not found"})
		return
	}

	user.IsActive = req.IsActive

	if err := h.userService.UpdateUser(user); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	status := "activated"
	if !req.IsActive {
		status = "deactivated"
	}

	c.JSON(http.StatusOK, gin.H{"message": "User " + status + " successfully"})
}
