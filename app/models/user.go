package models

import (
	"encoding/json"
	"time"
)

type User struct {
	ID           uint       `gorm:"primaryKey" json:"id"`
	Username     string     `gorm:"type:varchar(50);uniqueIndex;not null" json:"username"`
	FullName     string     `gorm:"type:varchar(100);not null" json:"full_name"`
	Role         string     `gorm:"type:varchar(20);default:'Staff'" json:"role"` // Admin | Staff
	PasswordHash string     `gorm:"type:varchar(256);not null" json:"-"`
	IsActive     bool       `gorm:"type:tinyint(1);default:1" json:"is_active"`
	CreatedAt    time.Time  `gorm:"default:CURRENT_TIMESTAMP(3)" json:"created_at"`
	LastLogin    *time.Time `json:"last_login"`
	Permissions  *string    `gorm:"type:text" json:"permissions"` // JSON array of module names
}

func (User) TableName() string { return "user" }

func (u *User) GetPermissions() []string {
	if u.Role == "Admin" {
		return nil // nil means all access
	}
	if u.Permissions == nil || *u.Permissions == "" {
		return []string{}
	}
	var perms []string
	_ = json.Unmarshal([]byte(*u.Permissions), &perms)
	return perms
}

func (u *User) CanAccess(module string) bool {
	if u.Role == "Admin" {
		return true
	}
	perms := u.GetPermissions()
	if len(perms) == 0 {
		return true // no restrictions = full access for staff with no explicit perms
	}
	for _, p := range perms {
		if p == module {
			return true
		}
	}
	return false
}

func (u *User) SetPermissions(modules []string) {
	if len(modules) == 0 {
		u.Permissions = nil
		return
	}
	b, _ := json.Marshal(modules)
	s := string(b)
	u.Permissions = &s
}

// AuditLog records every significant action in the system.
type AuditLog struct {
	ID         uint       `gorm:"primaryKey" json:"id"`
	Timestamp  time.Time  `gorm:"not null;index" json:"timestamp"`
	UserID     *uint      `gorm:"index" json:"user_id"`
	Username   string     `gorm:"type:varchar(80)" json:"username"`
	Action     string     `gorm:"type:varchar(30);not null" json:"action"` // CREATE | UPDATE | DELETE | LOGIN | LOGOUT | VOID | REVERSE
	Module     string     `gorm:"type:varchar(50);not null" json:"module"`
	EntityType string     `gorm:"type:varchar(50)" json:"entity_type"`
	EntityID   *uint      `json:"entity_id"`
	EntityRef  string     `gorm:"type:varchar(100)" json:"entity_ref"`
	Description string    `gorm:"type:text" json:"description"`
	IPAddress  string     `gorm:"type:varchar(45)" json:"ip_address"`
}

func (AuditLog) TableName() string { return "audit_log" }
