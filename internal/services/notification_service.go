package services

import (
	"log"
	"sync"
	"time"

	"leave-management-system/internal/models"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// NotificationService handles logic for notifications and Server-Sent Events (SSE).
type NotificationService struct {
	db *gorm.DB

	// clients maps a UserID to a specific channel of notification events.
	clients map[uuid.UUID]chan *models.Notification
	mutex   sync.RWMutex
}

// NewNotificationService creates a new notification service.
func NewNotificationService(db *gorm.DB) *NotificationService {
	return &NotificationService{
		db:      db,
		clients: make(map[uuid.UUID]chan *models.Notification),
	}
}

// AddClient registers a new SSE client for a specific user.
func (s *NotificationService) AddClient(userID uuid.UUID) chan *models.Notification {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	// Create a new channel with a small buffer.
	clientChan := make(chan *models.Notification, 100)
	s.clients[userID] = clientChan
	log.Printf("SSE Client added for user %s", userID)

	return clientChan
}

// RemoveClient unregisters an SSE client.
func (s *NotificationService) RemoveClient(userID uuid.UUID, clientChan chan *models.Notification) {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	// Optional: we can safely close it if we want, but letting garbage collection handle
	// unreferenced channels is also safe in Go if we remove the reference.
	if s.clients[userID] == clientChan {
		delete(s.clients, userID)
		close(clientChan)
		log.Printf("SSE Client removed for user %s", userID)
	}
}

// SendNotification creates a notification in the database and broadcasts it to the real-time client if connected.
func (s *NotificationService) SendNotification(userID uuid.UUID, title, message string, notifType models.NotificationType, relatedEntityID *uuid.UUID) error {
	notification := models.Notification{
		ID:              uuid.New(),
		UserID:          userID,
		Title:           title,
		Message:         message,
		Type:            notifType,
		IsRead:          false,
		RelatedEntityID: relatedEntityID,
		CreatedAt:       time.Now(),
		UpdatedAt:       time.Now(),
	}

	// 1. Save to Database
	if err := s.db.Create(&notification).Error; err != nil {
		return err
	}

	// 2. Broadcast to connected SSE client via channel
	s.mutex.RLock()
	clientChan, exists := s.clients[userID]
	s.mutex.RUnlock()

	if exists {
		// Non-blocking send wrapper
		select {
		case clientChan <- &notification:
			// log.Printf("Notification successfully pushed to user %s via SSE", userID)
		default:
			// Buffer full, drop SSE payload (user can still fetch it upon refresh)
			log.Printf("Warning: SSE buffer full for user %s, dropping push event.", userID)
		}
	}

	return nil
}

// GetUserNotifications returns paginated notifications for a user.
func (s *NotificationService) GetUserNotifications(userID uuid.UUID, limit, offset int) ([]models.Notification, error) {
	var notifications []models.Notification
	err := s.db.Where("user_id = ?", userID).
		Order("created_at DESC").
		Limit(limit).
		Offset(offset).
		Find(&notifications).Error
	return notifications, err
}

// GetUnreadCount returns the total number of unread notifications for a user.
func (s *NotificationService) GetUnreadCount(userID uuid.UUID) (int64, error) {
	var count int64
	err := s.db.Model(&models.Notification{}).
		Where("user_id = ? AND is_read = ?", userID, false).
		Count(&count).Error
	return count, err
}

// MarkAsRead marks a specific notification as read.
func (s *NotificationService) MarkAsRead(notificationID uuid.UUID, userID uuid.UUID) error {
	return s.db.Model(&models.Notification{}).
		Where("id = ? AND user_id = ?", notificationID, userID).
		Update("is_read", true).Error
}

// MarkAllAsRead marks all unread notifications for a user as read.
func (s *NotificationService) MarkAllAsRead(userID uuid.UUID) error {
	return s.db.Model(&models.Notification{}).
		Where("user_id = ? AND is_read = ?", userID, false).
		Update("is_read", true).Error
}
