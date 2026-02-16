package handlers

import (
	"leave-management-system/internal/models"
	"leave-management-system/internal/services"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type DelegationHandler struct {
	delegationSvc *services.DelegationService
	userSvc       *services.UserService
}

func NewDelegationHandler(delegationSvc *services.DelegationService, userSvc *services.UserService) *DelegationHandler {
	return &DelegationHandler{
		delegationSvc: delegationSvc,
		userSvc:       userSvc,
	}
}

// CreateDelegation creates a new delegation
func (h *DelegationHandler) CreateDelegation(c *gin.Context) {
	var req struct {
		DelegateID uuid.UUID `json:"delegate_id" binding:"required"`
		StartDate  time.Time `json:"start_date" binding:"required"`
		EndDate    time.Time `json:"end_date" binding:"required"`
		Reason     string    `json:"reason"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	userID := c.MustGet("user_id").(uuid.UUID)

	delegation := models.UserDelegation{
		DelegatorID: userID,
		DelegateID:  req.DelegateID,
		StartDate:   req.StartDate,
		EndDate:     req.EndDate,
		Reason:      req.Reason,
		Status:      "active",
	}

	if err := h.delegationSvc.CreateDelegation(&delegation); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, delegation)
}

// GetMyDelegations returns delegations created by the current user
func (h *DelegationHandler) GetMyDelegations(c *gin.Context) {
	userID := c.MustGet("user_id").(uuid.UUID)
	delegations, err := h.delegationSvc.GetDelegations(userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, delegations)
}

// CancelDelegation cancels a delegation
func (h *DelegationHandler) CancelDelegation(c *gin.Context) {
	idParam := c.Param("id")
	id, err := uuid.Parse(idParam)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid delegation ID"})
		return
	}

	userID := c.MustGet("user_id").(uuid.UUID)
	if err := h.delegationSvc.CancelDelegation(id, userID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Delegation cancelled"})
}

// GetDelegationCandidates searches for users eligible to be delegates
func (h *DelegationHandler) GetDelegationCandidates(c *gin.Context) {
	query := c.Query("q")
	if len(query) < 2 {
		c.JSON(http.StatusOK, []models.User{})
		return
	}

	users, err := h.userSvc.SearchUsers(query)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// Filter out self?
	userID := c.MustGet("user_id").(uuid.UUID)
	filtered := make([]models.User, 0)
	for _, u := range users {
		if u.ID != userID && u.IsActive {
			filtered = append(filtered, u)
		}
	}

	c.JSON(http.StatusOK, filtered)
}
