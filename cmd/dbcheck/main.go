package main

import (
	appconfig "egglayererp/app/config"
	"egglayererp/app/db"
	"fmt"
	"os"
)

func must(err error, step string) {
	if err != nil {
		fmt.Printf("FAIL [%s]: %v\n", step, err)
		os.Exit(1)
	}
	fmt.Printf("OK   [%s]\n", step)
}

func main() {
	cfg, err := appconfig.Load()
	must(err, "load config")
	must(db.Connect(cfg), "connect")
	must(db.PreMigrateFixup(), "PreMigrateFixup")
	must(db.AutoMigrateAll(), "AutoMigrateAll")
	must(db.RunColumnMigrations(), "RunColumnMigrations")
	must(db.SeedDefaults(), "SeedDefaults")
	fmt.Println("\n✓ All startup steps passed — login screen should appear.")
}
