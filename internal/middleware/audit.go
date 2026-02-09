package middleware

import (
	"bytes"
	"io"
	"leave-management-system/internal/models"
	"leave-management-system/internal/services"
	"leave-management-system/pkg/logger"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type AuditMiddleware struct {
	auditLogger  *logger.AuditLogger
	auditService *services.AuditService
}

func NewAuditMiddleware(auditLogger *logger.AuditLogger, auditService *services.AuditService) *AuditMiddleware {
	return &AuditMiddleware{
		auditLogger:  auditLogger,
		auditService: auditService,
	}
}

func (m *AuditMiddleware) AuditLog() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Skip logging for certain endpoints
		if strings.Contains(c.Request.URL.Path, "/health") ||
			strings.Contains(c.Request.URL.Path, "/metrics") {
			c.Next()
			return
		}

		// Capture request body
		var requestBody []byte
		if c.Request.Body != nil {
			requestBody, _ = io.ReadAll(c.Request.Body)
			c.Request.Body = io.NopCloser(bytes.NewBuffer(requestBody))
		}

		// Capture response
		blw := &bodyLogWriter{body: bytes.NewBufferString(""), ResponseWriter: c.Writer}
		c.Writer = blw

		startTime := time.Now()
		c.Next()
		duration := time.Since(startTime)

		// Get user info if authenticated
		var (
			uid   uuid.UUID
			email string
			role  models.UserRole
		)

		if val, ok := c.Get("user_id"); ok {
			uid = val.(uuid.UUID)
		}
		if val, ok := c.Get("user_email"); ok {
			email = val.(string)
		}
		if val, ok := c.Get("user_role"); ok {
			role = val.(models.UserRole)
		}

		// Log audit trail to file
		m.auditLogger.LogHTTP(
			c.Request.Method,
			c.Request.URL.Path,
			c.Writer.Status(),
			duration,
			c.ClientIP(),
			c.Request.UserAgent(),
			uid,
			email,
			role,
			string(requestBody),
			blw.body.String(),
		)

		// Save audit log to database (only for authenticated requests AND state-changing methods)
		// AND if not already logged by handler (deduplication)
		alreadyLogged, _ := c.Get("audit_logged")
		if uid != uuid.Nil && m.auditService != nil && alreadyLogged == nil {
			method := c.Request.Method
			// Only log state-changing methods to DB to reduce noise
			if method == "POST" || method == "PUT" || method == "DELETE" || method == "PATCH" {
				auditLog := &models.AuditLog{
					ID:         uuid.New(),
					ActorID:    uid,
					ActorEmail: email,
					ActorRole:  role,
					Action:     method + " " + c.Request.URL.Path,
					Method:     method,
					Endpoint:   c.Request.URL.Path,
					IPAddress:  c.ClientIP(),
					UserAgent:  c.Request.UserAgent(),
					CreatedAt:  time.Now(),
				}
				// Fire and forget - don't block request on audit log save
				go m.auditService.CreateAuditLog(auditLog)
			}
		}
	}
}

type bodyLogWriter struct {
	gin.ResponseWriter
	body *bytes.Buffer
}

func (w bodyLogWriter) Write(b []byte) (int, error) {
	w.body.Write(b)
	return w.ResponseWriter.Write(b)
}
