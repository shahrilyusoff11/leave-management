package main

import (
	"fmt"
	"time"
	"leave-management-system/internal/models"
	"leave-management-system/pkg/auth"
)

func main() {
	manager := auth.NewJWTManager("test-secret", 24*time.Hour)
	user := &models.User{
		Email: "test@example.com",
		Role:  models.RoleSysAdmin,
	}
	token, _, err := manager.Generate(user)
	if err != nil {
		fmt.Println("Generate Error:", err)
		return
	}
	fmt.Println("Generated Token:", token)
	
	claims, err := manager.Verify(token)
	if err != nil {
		fmt.Println("Verify Error:", err)
		return
	}
	fmt.Println("Verify Success:", claims.Email)
}
