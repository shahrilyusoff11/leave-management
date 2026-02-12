package database

import (
	"fmt"
	"leave-management-system/internal/models"
	"leave-management-system/internal/services"

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
		&models.Department{},
		&models.User{},
		&models.HODDelegation{},
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
	)

	if err != nil {
		return fmt.Errorf("failed to migrate database: %w", err)
	}

	// Create indexes
	db.Exec("CREATE INDEX IF NOT EXISTS idx_leave_requests_user_id ON leave_requests(user_id)")
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
	db.Exec("CREATE INDEX IF NOT EXISTS idx_hod_delegations_department_id ON hod_delegations(department_id)")
	db.Exec("CREATE INDEX IF NOT EXISTS idx_hod_delegations_delegate_id ON hod_delegations(delegate_id)")
	db.Exec("CREATE INDEX IF NOT EXISTS idx_hod_delegations_dates ON hod_delegations(start_date, end_date)")

	return nil
}
