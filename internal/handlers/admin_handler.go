package handlers

import (
	"encoding/json"
	"leave-management-system/internal/models"
	"leave-management-system/internal/services"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type AdminHandler struct {
	holidayService         *services.HolidayService
	configService          *services.ConfigService
	leaveService           *services.LeaveService
	auditService           *services.AuditService
	leaveTypeConfigService *services.LeaveTypeConfigService
}

func NewAdminHandler(holidayService *services.HolidayService,
	configService *services.ConfigService,
	leaveService *services.LeaveService,
	auditService *services.AuditService,
	leaveTypeConfigService *services.LeaveTypeConfigService) *AdminHandler {
	return &AdminHandler{
		holidayService:         holidayService,
		configService:          configService,
		leaveService:           leaveService,
		auditService:           auditService,
		leaveTypeConfigService: leaveTypeConfigService,
	}
}

type CreateHolidayRequest struct {
	Name        string    `json:"name" binding:"required"`
	Date        time.Time `json:"date" binding:"required"`
	Description string    `json:"description"`
	State       string    `json:"state"`
}

func (h *AdminHandler) CreatePublicHoliday(c *gin.Context) {
	var req CreateHolidayRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	holiday := &models.PublicHoliday{
		Name:        req.Name,
		Date:        req.Date,
		Description: req.Description,
		State:       req.State,
		IsActive:    true,
	}

	if err := h.holidayService.AddHoliday(holiday); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, holiday)
}

func (h *AdminHandler) GetPublicHolidays(c *gin.Context) {
	yearStr := c.Query("year")
	var year int

	if yearStr == "" {
		year = time.Now().Year()
	} else {
		var err error
		year, err = strconv.Atoi(yearStr)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid year"})
			return
		}
	}

	holidays, err := h.holidayService.GetHolidays(year)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, holidays)
}

func (h *AdminHandler) UpdatePublicHoliday(c *gin.Context) {
	holidayID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid holiday ID"})
		return
	}

	var updates map[string]interface{}
	if err := c.ShouldBindJSON(&updates); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if err := h.holidayService.UpdateHoliday(holidayID, updates); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Holiday updated"})
}

func (h *AdminHandler) DeletePublicHoliday(c *gin.Context) {
	holidayID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid holiday ID"})
		return
	}

	if err := h.holidayService.DeleteHoliday(holidayID); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Holiday deleted"})
}

type SystemConfigRequest struct {
	MaxCarryForwardDays int      `json:"max_carry_forward_days"`
	WorkingDays         []string `json:"working_days"`
	EscalationDays      int      `json:"escalation_days"`
}

func (h *AdminHandler) UpdateSystemConfig(c *gin.Context) {
	var req SystemConfigRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	svcReq := services.SystemConfigRequest{
		MaxCarryForwardDays: req.MaxCarryForwardDays,
		WorkingDays:         req.WorkingDays,
		EscalationDays:      req.EscalationDays,
	}

	if err := h.configService.UpdateSystemConfig(svcReq); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "System configuration updated"})
}

func (h *AdminHandler) GetSystemConfig(c *gin.Context) {
	config, err := h.configService.GetSystemConfig()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, config)
}

func (h *AdminHandler) TriggerYearEndProcess(c *gin.Context) {
	if err := h.leaveService.ProcessYearEndCarryForward(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Year-end process completed"})
}

func (h *AdminHandler) GetAuditLogs(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "50"))
	actorID := c.Query("actor_id")
	targetID := c.Query("target_id")
	startDate := c.Query("start_date")
	endDate := c.Query("end_date")

	logs, total, err := h.auditService.GetAuditLogs(page, limit, actorID, targetID, startDate, endDate)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"logs":  logs,
		"total": total,
		"page":  page,
		"limit": limit,
	})
}

// GetLeaveTypeConfigs returns all leave type configurations
func (h *AdminHandler) GetLeaveTypeConfigs(c *gin.Context) {
	configs, err := h.leaveTypeConfigService.GetAllConfigs()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, configs)
}

// UpdateLeaveTypeConfig updates configuration for a specific leave type
func (h *AdminHandler) UpdateLeaveTypeConfig(c *gin.Context) {
	leaveType := models.LeaveType(c.Param("type"))

	var updates map[string]interface{}
	if err := c.ShouldBindJSON(&updates); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// 1. Fetch Before State
	beforeConfig, err := h.leaveTypeConfigService.GetConfig(leaveType)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Leave type config not found"})
		return
	}

	// 2. Perform Update
	if err := h.leaveTypeConfigService.UpdateConfig(leaveType, updates); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// 3. Fetch After State
	afterConfig, err := h.leaveTypeConfigService.GetConfig(leaveType)
	if err != nil {
		// This shouldn't happen if update succeeded, but handle gracefully
		c.JSON(http.StatusOK, gin.H{"message": "Leave type configuration updated, but failed to fetch new state"})
		return
	}

	// 4. Audit Log (Fire and Forget)
	go func(before, after *models.LeaveTypeConfig, ctx *gin.Context) {
		// Extract user info from context (needs to be passed or extracted safely)
		// Note: passing *gin.Context to goroutine is unsafe. We should extract values first.
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
			Action:      "update_leave_type_config",
			TargetID:    before.ID,
			TargetType:  "leave_type_config",
			BeforeState: toMap(before),
			AfterState:  toMap(after),
			Method:      "PUT",
			Endpoint:    ctx.Request.URL.Path,
			IPAddress:   ctx.ClientIP(),
			UserAgent:   ctx.Request.UserAgent(),
			CreatedAt:   time.Now(),
		}
		h.auditService.CreateAuditLog(auditLog)
	}(beforeConfig, afterConfig, c.Copy()) // Use c.Copy() for goroutine safety

	// Prevent duplicate logging in middleware
	c.Set("audit_logged", true)

	c.JSON(http.StatusOK, gin.H{"message": "Leave type configuration updated"})
}

// GetAllWorkflows returns all workflow configurations
func (h *AdminHandler) GetAllWorkflows(c *gin.Context) {
	workflowSvc := services.NewWorkflowService(h.leaveService.GetDB())
	workflows, err := workflowSvc.GetAllWorkflows()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, workflows)
}

// GetWorkflow returns workflow for a specific leave type
func (h *AdminHandler) GetWorkflow(c *gin.Context) {
	leaveType := models.LeaveType(c.Param("type"))
	workflowSvc := services.NewWorkflowService(h.leaveService.GetDB())

	workflow, err := workflowSvc.GetWorkflowForLeaveType(leaveType)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Workflow not found"})
		return
	}

	c.JSON(http.StatusOK, workflow)
}

// UpdateWorkflow updates a workflow configuration
func (h *AdminHandler) UpdateWorkflow(c *gin.Context) {
	leaveType := models.LeaveType(c.Param("type"))
	workflowSvc := services.NewWorkflowService(h.leaveService.GetDB())

	workflow, err := workflowSvc.GetWorkflowForLeaveType(leaveType)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Workflow not found"})
		return
	}

	var updates struct {
		WorkflowName string `json:"workflow_name"`
		Description  string `json:"description"`
		IsActive     *bool  `json:"is_active"`
	}
	if err := c.ShouldBindJSON(&updates); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if updates.WorkflowName != "" {
		workflow.WorkflowName = updates.WorkflowName
	}
	if updates.Description != "" {
		workflow.Description = updates.Description
	}
	if updates.IsActive != nil {
		workflow.IsActive = *updates.IsActive
	}

	if err := workflowSvc.UpdateWorkflow(workflow); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, workflow)
}

// CreateWorkflowStep adds a new step to a workflow
func (h *AdminHandler) CreateWorkflowStep(c *gin.Context) {
	leaveType := models.LeaveType(c.Param("type"))
	workflowSvc := services.NewWorkflowService(h.leaveService.GetDB())

	workflow, err := workflowSvc.GetWorkflowForLeaveType(leaveType)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Workflow not found"})
		return
	}

	var step models.WorkflowStep
	if err := c.ShouldBindJSON(&step); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	step.WorkflowID = workflow.ID
	if err := workflowSvc.CreateWorkflowStep(&step); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, step)
}

// UpdateWorkflowStep updates an existing workflow step
func (h *AdminHandler) UpdateWorkflowStep(c *gin.Context) {
	stepID, err := uuid.Parse(c.Param("stepId"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid step ID"})
		return
	}

	workflowSvc := services.NewWorkflowService(h.leaveService.GetDB())

	step, err := workflowSvc.GetWorkflowStep(stepID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Step not found"})
		return
	}

	var updates models.WorkflowStep
	if err := c.ShouldBindJSON(&updates); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Update fields
	step.StepName = updates.StepName
	step.StepLabel = updates.StepLabel
	step.ResponsibleRole = updates.ResponsibleRole
	step.ActionType = updates.ActionType
	step.TimeoutDays = updates.TimeoutDays
	step.TimeoutAction = updates.TimeoutAction
	step.FallbackStepID = updates.FallbackStepID
	step.NextStepOnApprove = updates.NextStepOnApprove
	step.NextStepOnReject = updates.NextStepOnReject
	step.NotifyRoles = updates.NotifyRoles
	step.RequiresDocument = updates.RequiresDocument
	step.DocumentType = updates.DocumentType
	step.IsTerminal = updates.IsTerminal
	step.TerminalStatus = updates.TerminalStatus
	step.Conditions = updates.Conditions

	if err := workflowSvc.UpdateWorkflowStep(step); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, step)
}

// DeleteWorkflowStep removes a step from a workflow
func (h *AdminHandler) DeleteWorkflowStep(c *gin.Context) {
	stepID, err := uuid.Parse(c.Param("stepId"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid step ID"})
		return
	}

	workflowSvc := services.NewWorkflowService(h.leaveService.GetDB())
	if err := workflowSvc.DeleteWorkflowStep(stepID); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Step deleted"})
}

// ReorderWorkflowSteps updates the order of steps in a workflow
func (h *AdminHandler) ReorderWorkflowSteps(c *gin.Context) {
	leaveType := models.LeaveType(c.Param("type"))
	workflowSvc := services.NewWorkflowService(h.leaveService.GetDB())

	workflow, err := workflowSvc.GetWorkflowForLeaveType(leaveType)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Workflow not found"})
		return
	}

	var stepOrders map[string]int
	if err := c.ShouldBindJSON(&stepOrders); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Convert string UUIDs to uuid.UUID
	orders := make(map[uuid.UUID]int)
	for stepIDStr, order := range stepOrders {
		stepID, err := uuid.Parse(stepIDStr)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid step ID: " + stepIDStr})
			return
		}
		orders[stepID] = order
	}

	if err := workflowSvc.ReorderWorkflowSteps(workflow.ID, orders); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Steps reordered"})
}
