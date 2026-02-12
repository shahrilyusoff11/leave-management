package main

import (
	"fmt"
	"leave-management-system/internal/models"
	"log"
	"time"

	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

func main() {
	// Connect to database
	dsn := "host=localhost port=5432 user=lms_user password=lms_password dbname=leave_management_system sslmode=disable"
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		log.Fatal("Failed to connect to database:", err)
	}

	// Clear existing data
	db.Exec("TRUNCATE TABLE users CASCADE")
	db.Exec("TRUNCATE TABLE leave_balances CASCADE")
	db.Exec("TRUNCATE TABLE public_holidays CASCADE")

	// Create users
	hashedPassword, _ := bcrypt.GenerateFromPassword([]byte("Password@123"), bcrypt.DefaultCost)

	// Create CEO (SysAdmin)
	ceo := models.User{
		ID:           uuid.New(),
		Email:        "ceo@company.com",
		PasswordHash: string(hashedPassword),
		FirstName:    "John",
		LastName:     "Doe",
		Role:         models.RoleSysAdmin,
		Position:     "CEO",
		JoinedDate:   time.Now().AddDate(-5, 0, 0),
		IsConfirmed:  true,
		IsActive:     true,
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	}
	db.Create(&ceo)

	// Create HR Manager
	hrManager := models.User{
		ID:           uuid.New(),
		Email:        "hr@company.com",
		PasswordHash: string(hashedPassword),
		FirstName:    "Sarah",
		LastName:     "Chen",
		Role:         models.RoleHR,
		Position:     "HR Manager",
		ManagerID:    &ceo.ID,
		JoinedDate:   time.Now().AddDate(-3, 0, 0),
		IsConfirmed:  true,
		IsActive:     true,
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	}
	db.Create(&hrManager)

	// Create Department Manager
	deptManager := models.User{
		ID:           uuid.New(),
		Email:        "manager@company.com",
		PasswordHash: string(hashedPassword),
		FirstName:    "Michael",
		LastName:     "Tan",
		Role:         models.RoleManager,
		Position:     "Engineering Manager",
		ManagerID:    &ceo.ID,
		JoinedDate:   time.Now().AddDate(-2, 0, 0),
		IsConfirmed:  true,
		IsActive:     true,
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	}
	db.Create(&deptManager)

	// Create Staff
	staff := models.User{
		ID:           uuid.New(),
		Email:        "staff@company.com",
		PasswordHash: string(hashedPassword),
		FirstName:    "Ahmad",
		LastName:     "Bin Abdullah",
		Role:         models.RoleStaff,
		Position:     "Software Engineer",
		ManagerID:    &deptManager.ID,
		JoinedDate:   time.Now().AddDate(-1, 0, 0),
		IsConfirmed:  true,
		IsActive:     true,
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	}
	db.Create(&staff)

	// Create leave balances
	currentYear := time.Now().Year()

	// Staff annual leave
	staffAnnualBalance := models.LeaveBalance{
		ID:               uuid.New(),
		UserID:           staff.ID,
		LeaveType:        models.LeaveTypeAnnual,
		Year:             currentYear,
		TotalEntitlement: 8,
		Used:             2,
		CarriedForward:   0,
		Adjusted:         0,
		IsOverridden:     false,
		CreatedAt:        time.Now(),
		UpdatedAt:        time.Now(),
	}
	db.Create(&staffAnnualBalance)

	// Staff sick leave
	staffSickBalance := models.LeaveBalance{
		ID:               uuid.New(),
		UserID:           staff.ID,
		LeaveType:        models.LeaveTypeSick,
		Year:             currentYear,
		TotalEntitlement: 14,
		Used:             3,
		CarriedForward:   0,
		Adjusted:         0,
		IsOverridden:     false,
		CreatedAt:        time.Now(),
		UpdatedAt:        time.Now(),
	}
	db.Create(&staffSickBalance)

	// Add Malaysian public holidays for current year
	holidays := []models.PublicHoliday{
		{
			ID:          uuid.New(),
			Name:        "New Year's Day",
			Date:        time.Date(currentYear, 1, 1, 0, 0, 0, 0, time.UTC),
			Description: "First day of the year",
			IsActive:    true,
			CreatedAt:   time.Now(),
		},
		{
			ID:          uuid.New(),
			Name:        "Chinese New Year",
			Date:        time.Date(currentYear, 1, 28, 0, 0, 0, 0, time.UTC),
			Description: "First day of Chinese New Year",
			IsActive:    true,
			CreatedAt:   time.Now(),
		},
		{
			ID:          uuid.New(),
			Name:        "Labour Day",
			Date:        time.Date(currentYear, 5, 1, 0, 0, 0, 0, time.UTC),
			Description: "International Workers' Day",
			IsActive:    true,
			CreatedAt:   time.Now(),
		},
		{
			ID:          uuid.New(),
			Name:        "Hari Raya Aidilfitri",
			Date:        time.Date(currentYear, 6, 28, 0, 0, 0, 0, time.UTC),
			Description: "Eid al-Fitr",
			IsActive:    true,
			CreatedAt:   time.Now(),
		},
		{
			ID:          uuid.New(),
			Name:        "National Day",
			Date:        time.Date(currentYear, 8, 31, 0, 0, 0, 0, time.UTC),
			Description: "Malaysia's Independence Day",
			IsActive:    true,
			CreatedAt:   time.Now(),
		},
		{
			ID:          uuid.New(),
			Name:        "Christmas Day",
			Date:        time.Date(currentYear, 12, 25, 0, 0, 0, 0, time.UTC),
			Description: "Christmas celebration",
			IsActive:    true,
			CreatedAt:   time.Now(),
		},
	}

	for _, holiday := range holidays {
		db.Create(&holiday)
	}

	fmt.Println("Test data seeded successfully!")
	fmt.Println("\nTest users created:")
	fmt.Println("1. CEO (SysAdmin): ceo@company.com / Password@123")
	fmt.Println("2. HR Manager: hr@company.com / Password@123")
	fmt.Println("3. Engineering Manager: manager@company.com / Password@123")
	fmt.Println("4. Staff: staff@company.com / Password@123")
	fmt.Println("\nDefault password for all users: Password@123")
}
