package main

import (
	"fmt"
	"os"
	"golang.org/x/crypto/bcrypt"
)

func main() {
	hash, err := bcrypt.GenerateFromPassword([]byte("admin123"), bcrypt.DefaultCost)
	if err != nil {
		panic(err)
	}
	sql := fmt.Sprintf("UPDATE `user` SET password_hash='%s' WHERE username='admin';", string(hash))
	fmt.Println(sql)
	// Also write to file so it's easy to copy
	os.WriteFile("reset_admin.sql", []byte(sql+"\n"), 0644)
	fmt.Println("SQL also written to reset_admin.sql")
}
