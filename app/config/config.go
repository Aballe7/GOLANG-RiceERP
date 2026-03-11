package config

import (
	"os"
	"path/filepath"

	"github.com/BurntSushi/toml"
)

type DatabaseConfig struct {
	DSN          string `toml:"dsn"`
	MaxOpenConns int    `toml:"max_open_conns"`
	MaxIdleConns int    `toml:"max_idle_conns"`
}

type AppConfig struct {
	Debug   bool   `toml:"debug"`
	AppName string `toml:"app_name"`
}

type Config struct {
	Database DatabaseConfig `toml:"database"`
	App      AppConfig      `toml:"app"`
}

// Load reads config.toml from the same directory as the executable.
func Load() (*Config, error) {
	exe, err := os.Executable()
	if err != nil {
		return nil, err
	}
	cfgPath := filepath.Join(filepath.Dir(exe), "config.toml")

	// Fallback: try current working directory (useful during dev)
	if _, err := os.Stat(cfgPath); os.IsNotExist(err) {
		cfgPath = "config.toml"
	}

	var cfg Config
	if _, err := toml.DecodeFile(cfgPath, &cfg); err != nil {
		return nil, err
	}
	return &cfg, nil
}
