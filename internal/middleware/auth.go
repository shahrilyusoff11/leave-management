package middleware

import (
	"leave-management-system/pkg/auth"
	"net/http"
	"strings"

	"leave-management-system/internal/models"

	"github.com/gin-gonic/gin"
)

type AuthMiddleware struct {
	jwtManager *auth.JWTManager
}

func NewAuthMiddleware(jwtManager *auth.JWTManager) *AuthMiddleware {
	return &AuthMiddleware{jwtManager: jwtManager}
}

func (m *AuthMiddleware) Authenticate() gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Authorization header required"})
			c.Abort()
			return
		}

		parts := strings.Split(authHeader, " ")
		if len(parts) != 2 || parts[0] != "Bearer" {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid authorization format"})
			c.Abort()
			return
		}

		claims, err := m.jwtManager.Verify(parts[1])
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid or expired token"})
			c.Abort()
			return
		}

		c.Set("user_id", claims.UserID)
		c.Set("user_email", claims.Email)
		c.Set("user_role", claims.Role)
		c.Set("user_roles", claims.Roles)
		c.Next()
	}
}

func (m *AuthMiddleware) RequireRole(requiredRole models.UserRole) gin.HandlerFunc {
	return func(c *gin.Context) {
		if _, exists := c.Get("user_role"); !exists {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "User role not found"})
			c.Abort()
			return
		}

		if !contextHasRole(c, requiredRole) {
			c.JSON(http.StatusForbidden, gin.H{"error": "Insufficient permissions"})
			c.Abort()
			return
		}

		c.Next()
	}
}

func (m *AuthMiddleware) RequireAnyRole(requiredRoles []models.UserRole) gin.HandlerFunc {
	return func(c *gin.Context) {
		if _, exists := c.Get("user_role"); !exists {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "User role not found"})
			c.Abort()
			return
		}

		for _, requiredRole := range requiredRoles {
			if contextHasRole(c, requiredRole) {
				c.Next()
				return
			}
		}

		c.JSON(http.StatusForbidden, gin.H{"error": "Insufficient permissions"})
		c.Abort()
	}
}

func contextHasRole(c *gin.Context, requiredRole models.UserRole) bool {
	if roles, exists := c.Get("user_roles"); exists {
		switch assigned := roles.(type) {
		case []models.UserRole:
			for _, role := range assigned {
				if role == requiredRole {
					return true
				}
			}
		case []interface{}:
			for _, role := range assigned {
				if role == requiredRole || role == string(requiredRole) {
					return true
				}
			}
		}
	}

	role, exists := c.Get("user_role")
	return exists && role == requiredRole
}
