package handlers

import (
	"leave-management-system/internal/models"
	"leave-management-system/internal/services"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type BlackoutDateHandler struct {
	service *services.BlackoutDateService
}

func NewBlackoutDateHandler(service *services.BlackoutDateService) *BlackoutDateHandler {
	return &BlackoutDateHandler{service: service}
}

func (h *BlackoutDateHandler) CreateBlackoutDate(c *gin.Context) {
	var req models.BlackoutDate
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body"})
		return
	}

	actorID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
		return
	}
	req.CreatedBy = actorID.(uuid.UUID)

	if err := h.service.CreateBlackoutDate(&req); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, req)
}

func (h *BlackoutDateHandler) UpdateBlackoutDate(c *gin.Context) {
	idParam := c.Param("id")
	id, err := uuid.Parse(idParam)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid blackout date ID"})
		return
	}

	var req models.BlackoutDate
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body"})
		return
	}

	if err := h.service.UpdateBlackoutDate(id, &req); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Blackout date updated successfully"})
}

func (h *BlackoutDateHandler) DeleteBlackoutDate(c *gin.Context) {
	idParam := c.Param("id")
	id, err := uuid.Parse(idParam)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid blackout date ID"})
		return
	}

	if err := h.service.DeleteBlackoutDate(id); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Blackout date deleted successfully"})
}

func (h *BlackoutDateHandler) GetBlackoutDates(c *gin.Context) {
	dates, err := h.service.GetBlackoutDates()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, dates)
}
