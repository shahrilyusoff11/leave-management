package services

import (
	"leave-management-system/internal/models"
	"testing"
	"time"

	"github.com/glebarez/sqlite"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"gorm.io/gorm"
)

func TestDelegationFlow(t *testing.T) {
	// Setup In-Memory DB
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	assert.NoError(t, err)
	db.AutoMigrate(&models.User{}, &models.Department{}, &models.LeaveRequest{}, &models.LeaveRequestWorkflowState{}, &models.WorkflowStep{}, &models.UserDelegation{}, &models.HODDelegation{})

	// Setup Services
	deptSvc := NewDepartmentService(db)
	delegationSvc := NewDelegationService(db)
	repoSvc := NewWorkflowService(db, deptSvc, delegationSvc)

	// Seed Data
	managerA_ID := uuid.New()
	managerA := models.User{ID: managerA_ID, Email: "managerA@example.com", Role: models.RoleManager, IsActive: true, FirstName: "Manager", LastName: "A"}
	db.Create(&managerA)

	managerB_ID := uuid.New()
	managerB := models.User{ID: managerB_ID, Email: "managerB@example.com", Role: models.RoleManager, IsActive: true, FirstName: "Manager", LastName: "B"}
	db.Create(&managerB)

	staffC_ID := uuid.New()
	staffC := models.User{ID: staffC_ID, Email: "staffC@example.com", Role: models.RoleStaff, IsActive: true, ManagerID: &managerA_ID, FirstName: "Staff", LastName: "C"}
	db.Create(&staffC)

	// Helper to create request and get responsible users
	checkResponsible := func(t *testing.T) []models.User {
		reqID := uuid.New()
		// Use LeaveTypeAnnual as per previous fix
		req := models.LeaveRequest{ID: reqID, UserID: staffC_ID, LeaveType: models.LeaveTypeAnnual}
		db.Create(&req)

		stepID := uuid.New()
		step := models.WorkflowStep{ID: stepID, ResponsibleRole: models.RoleManager}

		state := models.LeaveRequestWorkflowState{
			LeaveRequestID: reqID,
			CurrentStep:    &step,
		}

		users, err := repoSvc.GetResponsibleUsers(&state)
		assert.NoError(t, err)
		return users
	}

	t.Run("No Delegation -> Routes to Manager A", func(t *testing.T) {
		users := checkResponsible(t)
		assert.Len(t, users, 1)
		assert.Equal(t, managerA_ID, users[0].ID)
	})

	t.Run("Active Delegation A->B -> Routes to Manager B", func(t *testing.T) {
		// Create Active Delegation
		delegation := models.UserDelegation{
			ID:          uuid.New(),
			DelegatorID: managerA_ID,
			DelegateID:  managerB_ID,
			StartDate:   time.Now().Add(-24 * time.Hour), // Started yesterday
			EndDate:     time.Now().Add(24 * time.Hour),  // Ends tomorrow
			Status:      "active",
		}
		db.Create(&delegation)

		users := checkResponsible(t)
		assert.Len(t, users, 1)
		assert.Equal(t, managerB_ID, users[0].ID, "Should route to Delegate (Manager B)")

		// Cleanup for next test
		db.Delete(&delegation)
	})

	t.Run("Expired Delegation A->B -> Routes to Manager A", func(t *testing.T) {
		// Create Expired Delegation
		delegation := models.UserDelegation{
			ID:          uuid.New(),
			DelegatorID: managerA_ID,
			DelegateID:  managerB_ID,
			StartDate:   time.Now().Add(-48 * time.Hour),
			EndDate:     time.Now().Add(-24 * time.Hour), // Ended yesterday
			Status:      "active",                        // Status field says active, but dates say expired
		}
		db.Create(&delegation)

		users := checkResponsible(t)
		assert.Len(t, users, 1)
		assert.Equal(t, managerA_ID, users[0].ID, "Should ignore expired delegation and route to Manager A")

		// Cleanup
		db.Delete(&delegation)
	})

	t.Run("Future Delegation A->B -> Routes to Manager A", func(t *testing.T) {
		// Create Future Delegation
		delegation := models.UserDelegation{
			ID:          uuid.New(),
			DelegatorID: managerA_ID,
			DelegateID:  managerB_ID,
			StartDate:   time.Now().Add(24 * time.Hour), // Starts tomorrow
			EndDate:     time.Now().Add(48 * time.Hour),
			Status:      "active",
		}
		db.Create(&delegation)

		users := checkResponsible(t)
		assert.Len(t, users, 1)
		assert.Equal(t, managerA_ID, users[0].ID, "Should ignore future delegation and route to Manager A")
	})
}
