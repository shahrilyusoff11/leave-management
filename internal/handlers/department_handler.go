package handlers

import (
	"leave-management-system/internal/models"
	"leave-management-system/internal/services"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type DepartmentHandler struct {
	departmentSvc *services.DepartmentService
}

func NewDepartmentHandler(departmentSvc *services.DepartmentService) *DepartmentHandler {
	return &DepartmentHandler{departmentSvc: departmentSvc}
}

// GetDepartments returns all departments
func (h *DepartmentHandler) GetDepartments(c *gin.Context) {
	departments, err := h.departmentSvc.GetAllDepartments()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch departments"})
		return
	}
	c.JSON(http.StatusOK, departments)
}

// GetDepartment returns a single department
func (h *DepartmentHandler) GetDepartment(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid department ID"})
		return
	}

	dept, err := h.departmentSvc.GetDepartment(id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Department not found"})
		return
	}
	c.JSON(http.StatusOK, dept)
}

// CreateDepartment creates a new department
func (h *DepartmentHandler) CreateDepartment(c *gin.Context) {
	var input struct {
		Name  string     `json:"name" binding:"required"`
		HODID *uuid.UUID `json:"hod_id"`
	}

	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	dept := models.Department{
		Name:      input.Name,
		HODID:     input.HODID,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	if err := h.departmentSvc.CreateDepartment(&dept); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, dept)
}

// UpdateDepartment updates a department
func (h *DepartmentHandler) UpdateDepartment(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid department ID"})
		return
	}

	dept, err := h.departmentSvc.GetDepartment(id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Department not found"})
		return
	}

	var input struct {
		Name  string     `json:"name"`
		HODID *uuid.UUID `json:"hod_id"`
	}

	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if input.Name != "" {
		dept.Name = input.Name
	}
	dept.HODID = input.HODID

	if err := h.departmentSvc.UpdateDepartment(dept); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, dept)
}

// DeleteDepartment deletes a department
func (h *DepartmentHandler) DeleteDepartment(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid department ID"})
		return
	}

	if err := h.departmentSvc.DeleteDepartment(id); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Department deleted"})
}

// CreateHODDelegation creates a new HOD delegation (Acting HOD)
func (h *DepartmentHandler) CreateHODDelegation(c *gin.Context) {
	deptID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid department ID"})
		return
	}

	var input struct {
		DelegatorID uuid.UUID `json:"delegator_id" binding:"required"`
		DelegateID  uuid.UUID `json:"delegate_id" binding:"required"`
		StartDate   string    `json:"start_date" binding:"required"`
		EndDate     string    `json:"end_date" binding:"required"`
		Reason      string    `json:"reason"`
	}

	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	startDate, err := time.Parse("2006-01-02", input.StartDate)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid start date format (use YYYY-MM-DD)"})
		return
	}

	endDate, err := time.Parse("2006-01-02", input.EndDate)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid end date format (use YYYY-MM-DD)"})
		return
	}

	delegation := models.HODDelegation{
		DepartmentID: deptID,
		DelegatorID:  input.DelegatorID,
		DelegateID:   input.DelegateID,
		StartDate:    startDate,
		EndDate:      endDate,
		Reason:       input.Reason,
		IsActive:     true,
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	}

	if err := h.departmentSvc.CreateHODDelegation(&delegation); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, delegation)
}

// GetDelegations returns all delegations for a department
func (h *DepartmentHandler) GetDelegations(c *gin.Context) {
	deptID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid department ID"})
		return
	}

	delegations, err := h.departmentSvc.GetDelegationsForDepartment(deptID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch delegations"})
		return
	}

	c.JSON(http.StatusOK, delegations)
}

// DeleteHODDelegation removes a delegation
func (h *DepartmentHandler) DeleteHODDelegation(c *gin.Context) {
	delegationID, err := uuid.Parse(c.Param("delegationId"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid delegation ID"})
		return
	}

	if err := h.departmentSvc.DeleteHODDelegation(delegationID); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Delegation deleted"})
}
