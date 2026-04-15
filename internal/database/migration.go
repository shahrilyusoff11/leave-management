package database

import (
	"fmt"
	"leave-management-system/internal/models"
	"leave-management-system/internal/services"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

func Migrate(db *gorm.DB) error {
	// Enable UUID extension for PostgreSQL
	if err := db.Exec("CREATE EXTENSION IF NOT EXISTS \"uuid-ossp\"").Error; err != nil {
		fmt.Printf("Note: UUID extension might not be available: %v\n", err)
	}

	// Run migrations
	err := db.AutoMigrate(
		// Department must come before User (FK dependency)
		&models.Role{},
		&models.Department{},
		&models.User{},
		&models.HODDelegation{},
		&models.UserDelegation{},
		&models.LeaveRequest{},
		&models.LeaveBalance{},
		&models.Chronology{},
		&models.PublicHoliday{},
		&models.LeaveTypeConfig{},
		&models.AuditLog{},
		&services.SystemConfig{},
		// Workflow models - LeaveWorkflow must come before WorkflowStep (FK dependency)
		&models.LeaveWorkflow{},
		&models.WorkflowStep{},
		&models.LeaveRequestWorkflowState{},
		&models.BlackoutDate{},
		&models.Notification{},
	)

	if err != nil {
		return fmt.Errorf("failed to migrate database: %w", err)
	}

	if err := seedRoles(db); err != nil {
		return fmt.Errorf("failed to seed roles: %w", err)
	}

	if err := migrateUserRoles(db); err != nil {
		return fmt.Errorf("failed to migrate user roles: %w", err)
	}

	// Create indexes
	db.Exec("CREATE INDEX IF NOT EXISTS idx_leave_requests_user_id ON leave_requests(user_id)")
	db.Exec("CREATE INDEX IF NOT EXISTS idx_notifications_user_id ON notifications(user_id)")
	db.Exec("CREATE INDEX IF NOT EXISTS idx_leave_requests_status ON leave_requests(status)")
	db.Exec("CREATE INDEX IF NOT EXISTS idx_leave_requests_approver_id ON leave_requests(approver_id)")
	db.Exec("CREATE INDEX IF NOT EXISTS idx_leave_balance_user_year ON leave_balances(user_id, year)")
	db.Exec("CREATE INDEX IF NOT EXISTS idx_audit_logs_actor_id ON audit_logs(actor_id)")
	db.Exec("CREATE INDEX IF NOT EXISTS idx_audit_logs_created_at ON audit_logs(created_at)")
	// Workflow indexes
	db.Exec("CREATE INDEX IF NOT EXISTS idx_workflow_steps_workflow_id ON workflow_steps(workflow_id)")
	db.Exec("CREATE INDEX IF NOT EXISTS idx_workflow_steps_order ON workflow_steps(workflow_id, step_order)")
	db.Exec("CREATE INDEX IF NOT EXISTS idx_leave_workflows_leave_type ON leave_workflows(leave_type)")
	db.Exec("CREATE INDEX IF NOT EXISTS idx_workflow_states_request_id ON leave_request_workflow_states(leave_request_id)")
	// Department and delegation indexes
	db.Exec("CREATE INDEX IF NOT EXISTS idx_users_department_id ON users(department_id)")
	db.Exec("CREATE INDEX IF NOT EXISTS idx_users_role_id ON users(role_id)")
	db.Exec("CREATE INDEX IF NOT EXISTS idx_hod_delegations_department_id ON hod_delegations(department_id)")
	db.Exec("CREATE INDEX IF NOT EXISTS idx_hod_delegations_delegate_id ON hod_delegations(delegate_id)")
	db.Exec("CREATE INDEX IF NOT EXISTS idx_hod_delegations_dates ON hod_delegations(start_date, end_date)")

	return nil
}

func seedRoles(db *gorm.DB) error {
	now := time.Now()
	defaultRoles := []models.Role{
		{ID: uuid.New(), Name: models.RoleSysAdmin, CreatedAt: now, UpdatedAt: now},
		{ID: uuid.New(), Name: models.RoleAdmin, CreatedAt: now, UpdatedAt: now},
		{ID: uuid.New(), Name: models.RoleHR, CreatedAt: now, UpdatedAt: now},
		{ID: uuid.New(), Name: models.RoleManager, CreatedAt: now, UpdatedAt: now},
		{ID: uuid.New(), Name: models.RoleHOD, CreatedAt: now, UpdatedAt: now},
		{ID: uuid.New(), Name: models.RoleStaff, CreatedAt: now, UpdatedAt: now},
	}

	for _, role := range defaultRoles {
		if err := db.Where("name = ?", role.Name).FirstOrCreate(&role).Error; err != nil {
			return err
		}
	}

	return nil
}

func migrateUserRoles(db *gorm.DB) error {
	if err := db.Exec("ALTER TABLE users ADD COLUMN IF NOT EXISTS role_id uuid").Error; err != nil {
		return err
	}

	if legacyRoleColumnExists(db) {
		if err := db.Exec(`
			UPDATE users
			SET role_id = roles.id
			FROM roles
			WHERE users.role_id IS NULL
			  AND users.role = roles.name
		`).Error; err != nil {
			return err
		}
	}

	if err := db.Exec("ALTER TABLE users ALTER COLUMN role_id SET NOT NULL").Error; err != nil {
		return err
	}

	if legacyRoleColumnExists(db) {
		if err := db.Exec("ALTER TABLE users DROP COLUMN IF EXISTS role").Error; err != nil {
			return err
		}
	}

	return nil
}

func legacyRoleColumnExists(db *gorm.DB) bool {
	var count int64
	if err := db.Raw(`
		SELECT COUNT(*)
		FROM information_schema.columns
		WHERE table_name = 'users' AND column_name = 'role'
	`).Scan(&count).Error; err != nil {
		return false
	}

	return count > 0
}
