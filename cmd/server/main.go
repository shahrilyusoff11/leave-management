package main

import (
	"context"
	"leave-management-system/internal/config"
	"leave-management-system/internal/cron"
	"leave-management-system/internal/database"
	"leave-management-system/internal/handlers"
	"leave-management-system/internal/middleware"
	"leave-management-system/internal/services"
	"leave-management-system/pkg/auth"
	"leave-management-system/pkg/logger"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"leave-management-system/internal/models"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"go.uber.org/zap"
	"golang.org/x/crypto/bcrypt"
)

func main() {
	// Initialize logger
	appLogger := logger.NewLogger("development")
	defer appLogger.Sync()

	// Load configuration
	cfg, err := config.LoadConfig(appLogger)
	if err != nil {
		appLogger.Fatal("Failed to load config", zap.Error(err))
	}
	appLogger.Info("Loaded configuration",
		zap.String("JWT_TTL", cfg.JWT.AccessTokenTTL.String()),
		zap.String("Env", cfg.Server.Env),
	)

	// Initialize database
	db, err := database.NewDatabase(&cfg.Database, logger.NewGormLogger(appLogger))
	if err != nil {
		appLogger.Fatal("Failed to connect to database", zap.Error(err))
	}
	defer db.Close()

	// Run migrations
	if err := database.Migrate(db.DB); err != nil {
		appLogger.Fatal("Failed to migrate database", zap.Error(err))
	}

	// Create super admin if not exists
	hashedPassword, _ := bcrypt.GenerateFromPassword([]byte("P@ssw0rd"), bcrypt.DefaultCost)
	db.CreateSuperAdmin("superadmin@example.com", string(hashedPassword), "Super", "Admin")

	// Initialize services
	jwtManager := auth.NewJWTManager(cfg.JWT.SecretKey, cfg.JWT.AccessTokenTTL)
	auditLogger := logger.NewAuditLogger(appLogger)

	holidayService := services.NewHolidayService(db.DB)
	leaveTypeConfigService := services.NewLeaveTypeConfigService(db.DB)

	// Seed default leave type configs if none exist
	if err := leaveTypeConfigService.SeedDefaultConfigs(); err != nil {
		appLogger.Error("Failed to seed leave type configs", zap.Error(err))
	}

	// Seed default workflows if none exist
	workflowService := services.NewWorkflowService(db.DB)
	if err := workflowService.SeedDefaultWorkflows(); err != nil {
		appLogger.Error("Failed to seed default workflows", zap.Error(err))
	}

	leaveCalculator := services.NewLeaveCalculator(holidayService, leaveTypeConfigService)
	leaveService := services.NewLeaveService(db.DB, leaveCalculator, auditLogger, holidayService, leaveTypeConfigService)
	userService := services.NewUserService(db.DB, auditLogger, leaveTypeConfigService, leaveCalculator)
	configService := services.NewConfigService(db.DB) // Initialize config service with DB
	auditService := services.NewAuditService(db.DB)   // Initialize audit service with DB

	emailService := services.NewEmailService(
		cfg.Email.Host,
		cfg.Email.Port,
		cfg.Email.Username,
		cfg.Email.Password,
		cfg.Email.From,
	)

	// Initialize cron jobs
	cronJobs := cron.NewCronJobs(leaveService, emailService, workflowService, appLogger)
	if err := cronJobs.Start(); err != nil {
		appLogger.Error("Failed to start cron jobs", zap.Error(err))
	}
	defer cronJobs.Stop()

	// Initialize handlers
	authHandler := handlers.NewAuthHandler(userService, jwtManager)
	leaveHandler := handlers.NewLeaveHandler(leaveService)
	hrHandler := handlers.NewHRHandler(userService, leaveService, auditService)

	adminHandler := handlers.NewAdminHandler(holidayService, configService, leaveService, auditService, leaveTypeConfigService)
	uploadHandler := handlers.NewUploadHandler()

	// Initialize middleware
	authMiddleware := middleware.NewAuthMiddleware(jwtManager)
	auditMiddleware := middleware.NewAuditMiddleware(auditLogger, auditService)

	// Setup Gin router
	if cfg.Server.Env == "production" {
		gin.SetMode(gin.ReleaseMode)
	}

	router := gin.New()
	router.Use(gin.Recovery())
	router.Use(auditMiddleware.AuditLog())

	// Static files
	router.Static("/uploads", "./uploads")

	// Public routes
	public := router.Group("/api/v1")
	{
		public.POST("/login", authHandler.Login)
		public.GET("/health", func(c *gin.Context) {
			c.JSON(http.StatusOK, gin.H{"status": "healthy"})
		})
	}

	// Protected routes
	protected := router.Group("/api/v1")
	protected.Use(authMiddleware.Authenticate())
	{
		// Profile
		protected.GET("/profile", func(c *gin.Context) {
			userID := c.MustGet("user_id").(uuid.UUID)
			user, err := userService.GetUser(userID)
			if err != nil {
				c.JSON(http.StatusNotFound, gin.H{"error": "User not found"})
				return
			}
			c.JSON(http.StatusOK, user)
		})
		protected.PUT("/profile", authHandler.UpdateProfile)
		protected.PUT("/change-password", authHandler.ChangePassword)

		// Leave requests
		protected.POST("/leave-requests", leaveHandler.CreateLeaveRequest)
		protected.GET("/leave-requests", leaveHandler.GetMyLeaveRequests)
		protected.GET("/leave-requests/:id", leaveHandler.GetLeaveRequest)
		protected.GET("/leave-requests/:id/chronology", leaveHandler.GetLeaveRequestChronology)
		protected.GET("/leave-requests/:id/workflow-state", leaveHandler.GetWorkflowState)
		protected.PUT("/leave-requests/:id/cancel", leaveHandler.CancelLeaveRequest)

		protected.GET("/leave-balance", leaveHandler.GetLeaveBalance)
		protected.POST("/upload", uploadHandler.UploadFile)

		// Public holidays (accessible by all authenticated users for leave calculation)
		protected.GET("/public-holidays", adminHandler.GetPublicHolidays)
		// Leave type configs (accessible by all authenticated users)
		protected.GET("/leave-type-configs", adminHandler.GetLeaveTypeConfigs)

		// Manager routes
		manager := protected.Group("")
		manager.Use(authMiddleware.RequireAnyRole([]models.UserRole{
			models.RoleManager, models.RoleHOD, models.RoleHR, models.RoleAdmin, models.RoleSysAdmin,
		}))
		{
			manager.GET("/team/leave-requests", leaveHandler.GetTeamLeaveRequests)
			manager.PUT("/leave-requests/:id/approve", leaveHandler.ApproveLeaveRequest)
			manager.PUT("/leave-requests/:id/reject", leaveHandler.RejectLeaveRequest)
			manager.POST("/leave-requests/:id/workflow-action", leaveHandler.PerformWorkflowAction)
			manager.POST("/leave-requests/:id/convert", leaveHandler.ConvertLeaveType)
		}

		// HR routes
		hr := protected.Group("/hr")
		hr.Use(authMiddleware.RequireAnyRole([]models.UserRole{
			models.RoleHR, models.RoleAdmin, models.RoleSysAdmin,
		}))
		{
			hr.GET("/users", hrHandler.GetAllUsers)
			hr.GET("/users/:id", hrHandler.GetUser)
			hr.POST("/users", hrHandler.CreateUser)
			hr.PUT("/users/:id", hrHandler.UpdateUser)
			hr.PUT("/users/:id/status", hrHandler.ToggleUserActive)
			hr.PUT("/users/:id/probation", hrHandler.ConfirmProbation)
			hr.PUT("/users/:id/leave-balance", hrHandler.UpdateLeaveBalance)
			hr.GET("/leave-requests", hrHandler.GetLeaveRequests)
			hr.GET("/payroll-report", hrHandler.ExportPayrollReport)
		}

		// Admin routes
		admin := protected.Group("/admin")
		admin.Use(authMiddleware.RequireAnyRole([]models.UserRole{
			models.RoleAdmin, models.RoleSysAdmin,
		}))
		{
			admin.POST("/holidays", adminHandler.CreatePublicHoliday)
			admin.GET("/holidays", adminHandler.GetPublicHolidays)
			admin.PUT("/holidays/:id", adminHandler.UpdatePublicHoliday)
			admin.DELETE("/holidays/:id", adminHandler.DeletePublicHoliday)
			admin.GET("/config", adminHandler.GetSystemConfig)
			admin.PUT("/config", adminHandler.UpdateSystemConfig)
			admin.POST("/year-end-process", adminHandler.TriggerYearEndProcess)
			admin.GET("/audit-logs", adminHandler.GetAuditLogs)
			admin.GET("/leave-type-configs", adminHandler.GetLeaveTypeConfigs)
			admin.PUT("/leave-type-configs/:type", adminHandler.UpdateLeaveTypeConfig)
			// Workflow routes
			admin.GET("/workflows", adminHandler.GetAllWorkflows)
			admin.GET("/workflows/:type", adminHandler.GetWorkflow)
			admin.PUT("/workflows/:type", adminHandler.UpdateWorkflow)
			admin.POST("/workflows/:type/steps", adminHandler.CreateWorkflowStep)
			admin.PUT("/workflows/:type/steps/:stepId", adminHandler.UpdateWorkflowStep)
			admin.DELETE("/workflows/:type/steps/:stepId", adminHandler.DeleteWorkflowStep)
			admin.PUT("/workflows/:type/reorder", adminHandler.ReorderWorkflowSteps)
		}

		// SysAdmin routes
		sysadmin := protected.Group("/sysadmin")
		sysadmin.Use(authMiddleware.RequireRole(models.RoleSysAdmin))
		{
			// System maintenance endpoints
			sysadmin.POST("/force-escalation-check", func(c *gin.Context) {
				cronJobs.TriggerEscalationCheck()
				c.JSON(http.StatusOK, gin.H{"message": "Escalation check triggered"})
			})
			sysadmin.GET("/system-metrics", func(c *gin.Context) {
				// Return system metrics
				c.JSON(http.StatusOK, gin.H{"metrics": "system_metrics_here"})
			})
		}
	}

	// Start server
	srv := &http.Server{
		Addr:         ":" + cfg.Server.Port,
		Handler:      router,
		ReadTimeout:  cfg.Server.ReadTimeout,
		WriteTimeout: cfg.Server.WriteTimeout,
		IdleTimeout:  cfg.Server.IdleTimeout,
	}

	// Graceful shutdown
	go func() {
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			appLogger.Fatal("Failed to start server", zap.Error(err))
		}
	}()

	appLogger.Info("Server started", zap.String("port", cfg.Server.Port))

	// Wait for interrupt signal
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	appLogger.Info("Shutting down server...")

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		appLogger.Fatal("Server forced to shutdown", zap.Error(err))
	}

	appLogger.Info("Server exited properly")
}
