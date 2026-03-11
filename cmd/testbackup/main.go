package main

import (
	"egglayererp/app/config"
	backupsvc "egglayererp/app/services/backup"
	"fmt"
	"os"
)

func main() {
	fmt.Println("=== Backup Diagnostic ===")
	fmt.Println()

	// 1. Backup directory
	backupDir := backupsvc.GetBackupDir()
	fmt.Println("Backup dir:", backupDir)
	if _, err := os.Stat(backupDir); err != nil {
		fmt.Println("  ERROR: dir not accessible:", err)
	} else {
		fmt.Println("  Dir OK")
	}
	fmt.Println()

	// 2. Load config
	cfg, err := config.Load()
	if err != nil {
		fmt.Println("ERROR loading config.toml:", err)
		os.Exit(1)
	}
	fmt.Println("DSN:", cfg.Database.DSN[:20], "...")
	fmt.Println()

	// 3. Run backup
	destPath := backupDir + `\diag_test.sql`
	fmt.Println("Attempting backup to:", destPath)
	bf, err := backupsvc.CreateBackupToPath(cfg, destPath)
	if err != nil {
		fmt.Println()
		fmt.Println("=== BACKUP FAILED ===")
		fmt.Println(err.Error())
		os.Exit(1)
	}

	fmt.Println()
	fmt.Println("=== BACKUP SUCCEEDED ===")
	fmt.Printf("File: %s\nSize: %d bytes\nCreated: %s\n", bf.FileName, bf.FileSizeBytes, bf.CreatedAt)

	// Clean up
	_ = os.Remove(destPath)
}
