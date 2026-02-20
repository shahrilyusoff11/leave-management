package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"

	"leave-management-system/internal/services"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type NotificationHandler struct {
	notificationSvc *services.NotificationService
}

func NewNotificationHandler(notificationSvc *services.NotificationService) *NotificationHandler {
	return &NotificationHandler{
		notificationSvc: notificationSvc,
	}
}

// StreamSSE establishes an SSE connection for a user.
func (h *NotificationHandler) StreamSSE(c *gin.Context) {
	userIDStr, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	userID := userIDStr.(uuid.UUID)

	// Set necessary headers for SSE
	c.Writer.Header().Set("Content-Type", "text/event-stream")
	c.Writer.Header().Set("Cache-Control", "no-cache")
	c.Writer.Header().Set("Connection", "keep-alive")
	// Flush headers immediately
	c.Writer.Flush()

	// Register client
	clientChan := h.notificationSvc.AddClient(userID)
	defer h.notificationSvc.RemoveClient(userID, clientChan)

	// Send an initial connected ping
	fmt.Fprintf(c.Writer, "event: ping\ndata: connected\n\n")
	c.Writer.Flush()

	// Listen for notifications or client disconnect
	notify := c.Request.Context().Done()

	for {
		select {
		case <-notify:
			// Client disconnected
			return
		case notification := <-clientChan:
			// Marshal notification to JSON
			data, err := json.Marshal(notification)
			if err != nil {
				continue
			}

			// Format and send the SSE message
			fmt.Fprintf(c.Writer, "event: notification\ndata: %s\n\n", string(data))
			c.Writer.Flush()
		}
	}
}

// GetNotifications returns a user's notification history.
func (h *NotificationHandler) GetNotifications(c *gin.Context) {
	userID := c.MustGet("user_id").(uuid.UUID)

	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "20"))
	offset, _ := strconv.Atoi(c.DefaultQuery("offset", "0"))

	notifications, err := h.notificationSvc.GetUserNotifications(userID, limit, offset)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch notifications"})
		return
	}

	c.JSON(http.StatusOK, notifications)
}

// GetUnreadCount returns the number of unread notifications.
func (h *NotificationHandler) GetUnreadCount(c *gin.Context) {
	userID := c.MustGet("user_id").(uuid.UUID)

	count, err := h.notificationSvc.GetUnreadCount(userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get unread count"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"count": count})
}

// MarkAsRead marks a single notification as read.
func (h *NotificationHandler) MarkAsRead(c *gin.Context) {
	userID := c.MustGet("user_id").(uuid.UUID)

	notificationID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid notification ID"})
		return
	}

	if err := h.notificationSvc.MarkAsRead(notificationID, userID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to mark as read"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Marked as read"})
}

// MarkAllAsRead marks all of a user's notifications as read.
func (h *NotificationHandler) MarkAllAsRead(c *gin.Context) {
	userID := c.MustGet("user_id").(uuid.UUID)

	if err := h.notificationSvc.MarkAllAsRead(userID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to mark all as read"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "All marked as read"})
}
