package handlers

import (
	"egglayererp/app/db"
	"egglayererp/app/middleware"
	"egglayererp/app/models"
)

// GetFarmSettings returns all farm settings as a key-value map.
func GetFarmSettings() Response {
	if !middleware.Store.IsLoggedIn() {
		return unauthorized()
	}
	var settings []models.FarmSettings
	if err := db.DB.Find(&settings).Error; err != nil {
		return errResponse(err)
	}
	m := make(map[string]string, len(settings))
	for _, s := range settings {
		m[s.Key] = s.Value
	}
	return okResponse("", m)
}

// UpdateFarmSetting updates a single farm setting value.
func UpdateFarmSetting(key, value string) Response {
	if !middleware.Store.IsAdmin() {
		return unauthorized()
	}
	result := db.DB.Where("`key` = ?", key).
		Assign(models.FarmSettings{Key: key, Value: value}).
		FirstOrCreate(&models.FarmSettings{})
	if result.Error != nil {
		return errResponse(result.Error)
	}
	if err := db.DB.Model(&models.FarmSettings{}).Where("`key` = ?", key).
		Update("value", value).Error; err != nil {
		return errResponse(err)
	}
	return okResponse("Setting updated", nil)
}

// UpdateFarmSettings updates multiple settings at once.
func UpdateFarmSettings(settings map[string]string) Response {
	if !middleware.Store.IsAdmin() {
		return unauthorized()
	}
	for k, v := range settings {
		if err := db.DB.Exec(
			"INSERT INTO farm_settings (key, value) VALUES (?, ?) ON DUPLICATE KEY UPDATE value = ?",
			k, v, v,
		).Error; err != nil {
			return errResponse(err)
		}
	}
	return okResponse("Settings saved", nil)
}

// GetAuditLogs returns recent audit log entries.
func GetAuditLogs(limit int) Response {
	if !middleware.Store.IsAdmin() {
		return unauthorized()
	}
	if limit <= 0 {
		limit = 200
	}
	var logs []models.AuditLog
	if err := db.DB.Order("timestamp DESC").Limit(limit).Find(&logs).Error; err != nil {
		return errResponse(err)
	}
	return okResponse("", logs)
}
