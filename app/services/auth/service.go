package auth

import (
	"errors"
	"fmt"
	"time"

	"ricemill/app/db"
	"ricemill/app/models"

	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

// Login verifies credentials. Returns the user on success.
func Login(username, password string) (*models.User, error) {
	var user models.User
	if err := db.DB.Where("username = ?", username).First(&user).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, fmt.Errorf("invalid username or password")
		}
		return nil, err
	}
	if !user.IsActive {
		return nil, fmt.Errorf("account is inactive")
	}
	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(password)); err != nil {
		return nil, fmt.Errorf("invalid username or password")
	}
	now := time.Now()
	db.DB.Model(&user).Update("last_login", now)
	user.LastLogin = &now
	return &user, nil
}

// ListUsers returns all users.
func ListUsers() ([]models.User, error) {
	var users []models.User
	err := db.DB.Order("username").Find(&users).Error
	return users, err
}

// GetUser returns a user by ID.
func GetUser(id uint) (*models.User, error) {
	var user models.User
	err := db.DB.First(&user, id).Error
	return &user, err
}

// CreateUser creates a new user with a hashed password.
func CreateUser(username, fullName, role, password string) (*models.User, error) {
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return nil, err
	}
	user := models.User{
		Username:     username,
		FullName:     fullName,
		Role:         role,
		PasswordHash: string(hash),
		IsActive:     true,
		CreatedAt:    time.Now(),
	}
	if err := db.DB.Create(&user).Error; err != nil {
		return nil, err
	}
	return &user, nil
}

// UpdateUser updates user details (not password).
func UpdateUser(id uint, fullName, role string, isActive bool) error {
	return db.DB.Model(&models.User{}).Where("id = ?", id).Updates(map[string]interface{}{
		"full_name":  fullName,
		"role":       role,
		"is_active":  isActive,
	}).Error
}

// ChangePassword updates a user's password.
func ChangePassword(id uint, newPassword string) error {
	hash, err := bcrypt.GenerateFromPassword([]byte(newPassword), bcrypt.DefaultCost)
	if err != nil {
		return err
	}
	return db.DB.Model(&models.User{}).Where("id = ?", id).
		Update("password_hash", string(hash)).Error
}

// SetPermissions sets module permissions for a user.
func SetPermissions(id uint, modules []string) error {
	var user models.User
	if err := db.DB.First(&user, id).Error; err != nil {
		return err
	}
	user.SetPermissions(modules)
	return db.DB.Model(&user).Update("permissions", user.Permissions).Error
}

// DeleteUser soft-deletes (deactivates) a user.
func DeleteUser(id uint) error {
	return db.DB.Model(&models.User{}).Where("id = ?", id).
		Update("is_active", false).Error
}
