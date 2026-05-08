package services

import (
	"errors"
	"leave-management-system/internal/models"
	"leave-management-system/pkg/logger"
	"time"

	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

type UserService struct {
	db                 *gorm.DB
	auditLogger        *logger.AuditLogger
	leaveTypeConfigSvc *LeaveTypeConfigService
	leaveCalculator    *LeaveCalculator
}

func NewUserService(db *gorm.DB, auditLogger *logger.AuditLogger, leaveTypeConfigSvc *LeaveTypeConfigService, leaveCalculator *LeaveCalculator) *UserService {
	return &UserService{
		db:                 db,
		auditLogger:        auditLogger,
		leaveTypeConfigSvc: leaveTypeConfigSvc,
		leaveCalculator:    leaveCalculator,
	}
}

func (us *UserService) GetUserByEmail(email string) (*models.User, error) {
	var user models.User
	err := us.db.Preload("Roles").Where("email = ?", email).First(&user).Error
	return &user, err
}

func (us *UserService) GetUser(id uuid.UUID) (*models.User, error) {
	var user models.User
	err := us.db.Preload("Roles").First(&user, "id = ?", id).Error
	return &user, err
}

func (us *UserService) GetUserWithDetails(id uuid.UUID) (*models.User, error) {
	var user models.User
	err := us.db.Preload("Manager").
		Preload("Roles").
		Preload("DepartmentRef").
		Preload("LeaveEntitlements").
		Preload("ManagedUsers").
		First(&user, "id = ?", id).Error
	return &user, err
}

func (us *UserService) GetAllUsers() ([]models.User, error) {
	var users []models.User
	err := us.db.Preload("Roles").Preload("Manager").Preload("DepartmentRef").Find(&users).Error
	return users, err
}

func (us *UserService) CreateUser(user *models.User) error {
	// Only set default password if no password hash was provided
	if user.PasswordHash == "" {
		hashedPassword, err := bcrypt.GenerateFromPassword([]byte("Default@123"), bcrypt.DefaultCost)
		if err != nil {
			return err
		}
		user.PasswordHash = string(hashedPassword)
	}

	// Set default values
	if user.JoinedDate.IsZero() {
		user.JoinedDate = time.Now()
	}

	return us.db.Transaction(func(tx *gorm.DB) error {
		if err := us.normalizeUserRoles(tx, user); err != nil {
			return err
		}

		if err := tx.Create(user).Error; err != nil {
			return err
		}

		// Determine the year for initial leave balances
		// If joined in the past, use current year. If future, use join year.
		initialBalanceYear := user.JoinedDate.Year()
		currentYear := time.Now().Year()
		if initialBalanceYear < currentYear {
			initialBalanceYear = currentYear
		}

		// Create default leave balances
		return us.createDefaultLeaveBalances(tx, user.ID, user.JoinedDate, initialBalanceYear)
	})
}

func (us *UserService) createDefaultLeaveBalances(tx *gorm.DB, userID uuid.UUID, joinedDate time.Time, year int) error {
	// Create annual leave balance
	annualEntitlement := us.leaveCalculator.CalculateAnnualLeaveEntitlement(joinedDate, year)
	annualBalance := models.LeaveBalance{
		ID:               uuid.New(),
		UserID:           userID,
		LeaveType:        models.LeaveTypeAnnual,
		Year:             year,
		TotalEntitlement: annualEntitlement,
		CreatedAt:        time.Now(),
		UpdatedAt:        time.Now(),
	}

	// Create sick leave balance
	sickEntitlement := us.leaveCalculator.CalculateSickLeaveEntitlement(joinedDate, year)
	sickBalance := models.LeaveBalance{
		ID:               uuid.New(),
		UserID:           userID,
		LeaveType:        models.LeaveTypeSick,
		Year:             year,
		TotalEntitlement: sickEntitlement,
		CreatedAt:        time.Now(),
		UpdatedAt:        time.Now(),
	}

	if err := tx.Create(&annualBalance).Error; err != nil {
		return err
	}

	if err := tx.Create(&sickBalance).Error; err != nil {
		return err
	}

	return nil
}

func (us *UserService) ChangePassword(userID uuid.UUID, currentPassword, newPassword string) error {
	return us.db.Transaction(func(tx *gorm.DB) error {
		var user models.User
		if err := tx.First(&user, "id = ?", userID).Error; err != nil {
			return err
		}

		// Verify current password
		if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(currentPassword)); err != nil {
			return errors.New("current password is incorrect")
		}

		// Hash new password
		hashedPassword, err := bcrypt.GenerateFromPassword([]byte(newPassword), bcrypt.DefaultCost)
		if err != nil {
			return err
		}

		// Update password
		user.PasswordHash = string(hashedPassword)
		user.UpdatedAt = time.Now()

		return tx.Save(&user).Error
	})
}

func (us *UserService) UpdateProbationStatus(userID uuid.UUID, isConfirmed bool, notes string) error {
	return us.db.Transaction(func(tx *gorm.DB) error {
		var user models.User
		if err := tx.First(&user, "id = ?", userID).Error; err != nil {
			return err
		}

		user.IsConfirmed = isConfirmed
		user.UpdatedAt = time.Now()

		if isConfirmed && user.ProbationEndDate == nil {
			now := time.Now()
			user.ProbationEndDate = &now
		}

		return tx.Save(&user).Error
	})
}

func (us *UserService) GetTeamMembers(managerID uuid.UUID) ([]models.User, error) {
	var users []models.User
	err := us.db.Where("manager_id = ?", managerID).Find(&users).Error
	return users, err
}

func (us *UserService) UpdateUser(user *models.User) error {
	user.UpdatedAt = time.Now()
	return us.db.Transaction(func(tx *gorm.DB) error {
		if err := us.normalizeUserRoles(tx, user); err != nil {
			return err
		}

		if len(user.Roles) > 0 {
			if err := tx.Model(user).Association("Roles").Replace(user.Roles); err != nil {
				return err
			}
		}

		return tx.Save(user).Error
	})
}

func (us *UserService) SearchUsers(query string) ([]models.User, error) {
	var users []models.User
	if query == "" {
		return nil, nil
	}
	search := "%" + query + "%"
	err := us.db.Where("first_name ILIKE ? OR last_name ILIKE ? OR email ILIKE ?", search, search, search).
		Limit(20).Find(&users).Error
	return users, err
}

func (us *UserService) normalizeUserRoles(tx *gorm.DB, user *models.User) error {
	requested := make([]models.UserRole, 0, len(user.Roles)+1)
	seen := map[models.UserRole]bool{}

	for _, role := range user.Roles {
		if role.Name == "" || seen[role.Name] {
			continue
		}
		seen[role.Name] = true
		requested = append(requested, role.Name)
	}

	if user.Role != "" && !seen[user.Role] {
		seen[user.Role] = true
		requested = append(requested, user.Role)
	}

	if len(requested) == 0 {
		return nil
	}

	roles := make([]models.Role, 0, len(requested))
	for _, roleName := range requested {
		role, err := models.FindOrCreateRole(tx, roleName)
		if err != nil {
			return err
		}
		roles = append(roles, *role)
	}

	user.Roles = roles
	user.Role = models.PrimaryRoleFromRoles(roles)
	return nil
}
