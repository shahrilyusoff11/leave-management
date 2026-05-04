package services

import (
	"leave-management-system/internal/models"
	"testing"

	"github.com/glebarez/sqlite"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"gorm.io/gorm"
)

func TestSmartRouting(t *testing.T) {
	// Setup In-Memory DB
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	assert.NoError(t, err)
	db.AutoMigrate(&models.Role{}, &models.User{}, &models.Department{}, &models.LeaveRequest{}, &models.LeaveRequestWorkflowState{}, &models.WorkflowStep{}, &models.UserDelegation{}, &models.HODDelegation{}, &models.Notification{})

	// Setup Services
	// Note: We need full services to test interaction, but mocks would be better.
	// Given the tight coupling, integration-style unit test is easier here.
	deptSvc := NewDepartmentService(db)
	delegationSvc := NewDelegationService(db)
	repoSvc := NewWorkflowService(db, deptSvc, delegationSvc)

	// Seed Data
	adminID := uuid.New()
	admin := models.User{ID: adminID, Email: "admin@example.com", Role: models.RoleAdmin, IsActive: true, FirstName: "Admin", LastName: "User"}
	db.Create(&admin)

	bossID := uuid.New()
	boss := models.User{ID: bossID, Email: "boss@example.com", Role: models.RoleManager, IsActive: true, FirstName: "Big", LastName: "Boss"}
	db.Create(&boss)

	hodID := uuid.New()
	hod := models.User{ID: hodID, Email: "hod@example.com", Role: models.RoleHOD, IsActive: true, ManagerID: &bossID, FirstName: "HOD", LastName: "Department"}
	db.Create(&hod)

	staffID := uuid.New()
	staff := models.User{ID: staffID, Email: "staff@example.com", Role: models.RoleStaff, IsActive: true, ManagerID: &hodID, FirstName: "Staff", LastName: "Member"}
	db.Create(&staff)

	// Create Department with HOD
	deptID := uuid.New()
	dept := models.Department{ID: deptID, Name: "Engineering", HODID: &hodID}
	db.Create(&dept)

	// Assign users to dept
	db.Model(&hod).Update("department_id", deptID)
	db.Model(&staff).Update("department_id", deptID)

	t.Run("Normal Staff -> HOD Approval", func(t *testing.T) {
		reqID := uuid.New()
		req := models.LeaveRequest{ID: reqID, UserID: staffID, LeaveType: models.LeaveTypeAnnual}
		db.Create(&req)

		// Workflow Step: HOD
		stepID := uuid.New()
		step := models.WorkflowStep{ID: stepID, ResponsibleRole: models.RoleHOD}

		state := models.LeaveRequestWorkflowState{
			LeaveRequestID: reqID,
			CurrentStep:    &step,
		}

		users, err := repoSvc.GetResponsibleUsers(&state)
		assert.NoError(t, err)
		assert.Len(t, users, 1)
		assert.Equal(t, hodID, users[0].ID, "Should route to HOD")
	})

	t.Run("HOD Approval Does Not Notify Unrelated Department HOD", func(t *testing.T) {
		otherHODID := uuid.New()
		otherHOD := models.User{ID: otherHODID, Email: "other-hod@example.com", Role: models.RoleHOD, IsActive: true, FirstName: "Other", LastName: "HOD"}
		db.Create(&otherHOD)

		otherDeptID := uuid.New()
		otherDept := models.Department{ID: otherDeptID, Name: "Finance", HODID: &otherHODID}
		db.Create(&otherDept)
		db.Model(&otherHOD).Update("department_id", otherDeptID)

		reqID := uuid.New()
		req := models.LeaveRequest{ID: reqID, UserID: staffID, LeaveType: models.LeaveTypeAnnual}
		db.Create(&req)

		stepID := uuid.New()
		step := models.WorkflowStep{ID: stepID, ResponsibleRole: models.RoleHOD}

		state := models.LeaveRequestWorkflowState{
			LeaveRequestID: reqID,
			CurrentStep:    &step,
		}

		users, err := repoSvc.GetResponsibleUsers(&state)
		assert.NoError(t, err)
		assert.Len(t, users, 1)
		assert.Equal(t, hodID, users[0].ID, "Should only route to applicant department HOD")
		assert.NotEqual(t, otherHODID, users[0].ID, "Should not route to unrelated department HOD")
	})

	t.Run("HOD Approval Uses Manager Chain When Staff Has No Department", func(t *testing.T) {
		noDeptManagerID := uuid.New()
		noDeptManager := models.User{ID: noDeptManagerID, Email: "nodept-manager@example.com", Role: models.RoleManager, IsActive: true, ManagerID: &hodID, FirstName: "NoDept", LastName: "Manager"}
		db.Create(&noDeptManager)

		noDeptStaffID := uuid.New()
		noDeptStaff := models.User{ID: noDeptStaffID, Email: "nodept-staff@example.com", Role: models.RoleStaff, IsActive: true, ManagerID: &noDeptManagerID, FirstName: "NoDept", LastName: "Staff"}
		db.Create(&noDeptStaff)

		reqID := uuid.New()
		req := models.LeaveRequest{ID: reqID, UserID: noDeptStaffID, LeaveType: models.LeaveTypeAnnual}
		db.Create(&req)

		stepID := uuid.New()
		step := models.WorkflowStep{ID: stepID, ResponsibleRole: models.RoleHOD}

		state := models.LeaveRequestWorkflowState{
			LeaveRequestID: reqID,
			CurrentStep:    &step,
		}

		users, err := repoSvc.GetResponsibleUsers(&state)
		assert.NoError(t, err)
		assert.Len(t, users, 1)
		assert.Equal(t, hodID, users[0].ID, "Should route to the HOD in the applicant's manager chain")
	})

	t.Run("HOD Applicant -> Routes to Manager (Bypass Self)", func(t *testing.T) {
		reqID := uuid.New()
		req := models.LeaveRequest{ID: reqID, UserID: hodID, LeaveType: models.LeaveTypeAnnual}
		db.Create(&req)

		// Workflow Step: HOD (The configured step is HOD)
		stepID := uuid.New()
		step := models.WorkflowStep{ID: stepID, ResponsibleRole: models.RoleHOD}

		state := models.LeaveRequestWorkflowState{
			LeaveRequestID: reqID,
			CurrentStep:    &step,
		}

		users, err := repoSvc.GetResponsibleUsers(&state)
		assert.NoError(t, err)
		assert.Len(t, users, 1)
		assert.Equal(t, bossID, users[0].ID, "Should route to Manager (Boss), ensuring HOD doesn't approve self")
	})

	t.Run("Boss Applicant (No Manager) -> Routes to Admin (Orphan Fallback)", func(t *testing.T) {
		reqID := uuid.New()
		req := models.LeaveRequest{ID: reqID, UserID: bossID, LeaveType: models.LeaveTypeAnnual}
		db.Create(&req)

		// Workflow Step: Manager (Configured step is Manager)
		stepID := uuid.New()
		step := models.WorkflowStep{ID: stepID, ResponsibleRole: models.RoleManager}

		state := models.LeaveRequestWorkflowState{
			LeaveRequestID: reqID,
			CurrentStep:    &step,
		}

		users, err := repoSvc.GetResponsibleUsers(&state)
		assert.NoError(t, err)
		// Should find Admins because Boss has no Manager
		// Should find Admins because Boss has no Manager
		foundAdmin := false
		for _, u := range users {
			if u.ID == adminID {
				foundAdmin = true
				break
			}
		}
		assert.True(t, foundAdmin, "Should fall back to Admin(s)")
	})

	t.Run("HR Applicant -> Routes to Other HR (Self Filter)", func(t *testing.T) {
		// Create another HR
		hr2ID := uuid.New()
		hr2 := models.User{ID: hr2ID, Email: "hr2@example.com", Role: models.RoleHR, IsActive: true}
		db.Create(&hr2)

		// Create Applicant HR
		hrApplicantID := uuid.New()
		hrApplicant := models.User{ID: hrApplicantID, Email: "hr_app@example.com", Role: models.RoleHR, IsActive: true}
		db.Create(&hrApplicant)

		reqID := uuid.New()
		req := models.LeaveRequest{ID: reqID, UserID: hrApplicantID, LeaveType: models.LeaveTypeAnnual}
		db.Create(&req)

		stepID := uuid.New()
		step := models.WorkflowStep{ID: stepID, ResponsibleRole: models.RoleHR}

		state := models.LeaveRequestWorkflowState{
			LeaveRequestID: reqID,
			CurrentStep:    &step,
		}

		users, err := repoSvc.GetResponsibleUsers(&state)
		assert.NoError(t, err)

		// Should show hr2, but NOT hrApplicant
		foundSelf := false
		foundOther := false
		for _, u := range users {
			if u.ID == hrApplicantID {
				foundSelf = true
			}
			if u.ID == hr2ID {
				foundOther = true
			}
		}

		assert.False(t, foundSelf, "Should not include self in role-based pool")
		assert.True(t, foundOther, "Should include other HR")
	})
}
