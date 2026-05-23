package handlers

import (
	appconfig "ricemill/app/config"
	"ricemill/app/middleware"
	"ricemill/app/services/audit"
	"ricemill/app/services/backup"
)

var appCfg *appconfig.Config

// SetConfig stores the app config reference for use in backup operations.
func SetConfig(cfg *appconfig.Config) {
	appCfg = cfg
}

// CreateBackup triggers a mysqldump backup.
func CreateBackup() Response {
	if !middleware.Store.IsAdmin() {
		return unauthorized()
	}
	if appCfg == nil {
		return errMsg("App configuration not loaded")
	}
	bf, err := backup.CreateBackup(appCfg)
	if err != nil {
		return errResponse(err)
	}
	audit.Log("CREATE", "Backup", "Backup", bf.FileName, "Database backup created: "+bf.FileName, nil)
	return okResponse("Backup created successfully", bf)
}

// ListBackups returns all available backup files.
func ListBackups() Response {
	if !middleware.Store.IsLoggedIn() {
		return unauthorized()
	}
	files, err := backup.ListBackups()
	if err != nil {
		return errResponse(err)
	}
	return okResponse("", files)
}

// DeleteBackup removes a backup file by name.
func DeleteBackup(name string) Response {
	if !middleware.Store.IsAdmin() {
		return unauthorized()
	}
	if err := backup.DeleteBackup(name); err != nil {
		return errResponse(err)
	}
	audit.Log("DELETE", "Backup", "Backup", name, "Database backup deleted: "+name, nil)
	return okResponse("Backup deleted", nil)
}
