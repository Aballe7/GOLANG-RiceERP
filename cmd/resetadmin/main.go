package main

import (
	"fmt"
	"log"

	appconfig "egglayererp/app/config"
	"egglayererp/app/db"

	"golang.org/x/crypto/bcrypt"
)

func main() {
	cfg, err := appconfig.Load()
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	if err := db.Connect(cfg); err != nil {
		log.Fatalf("Failed to connect to DB: %v", err)
	}

	hash, err := bcrypt.GenerateFromPassword([]byte("admin123"), bcrypt.DefaultCost)
	if err != nil {
		log.Fatalf("Failed to hash password: %v", err)
	}

	result := db.DB.Exec(
		"UPDATE `user` SET password_hash = ?, is_active = 1 WHERE username = 'admin'",
		string(hash),
	)
	if result.Error != nil {
		log.Fatalf("Failed to update: %v", result.Error)
	}
	if result.RowsAffected == 0 {
		// No existing admin — insert one
		result = db.DB.Exec(
			"INSERT INTO `user` (username, full_name, role, password_hash, is_active, created_at) VALUES ('admin', 'System Administrator', 'Admin', ?, 1, NOW())",
			string(hash),
		)
		if result.Error != nil {
			log.Fatalf("Failed to insert admin: %v", result.Error)
		}
		fmt.Println("Admin user created with password: admin123")
	} else {
		fmt.Println("Admin password reset to: admin123")
	}
}
