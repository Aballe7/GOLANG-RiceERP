package config

import (
	"bytes"
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

// Load reads config.toml from the executable directory or CWD.
// If no external file is found, it falls back to the embedded defaultConfig bytes.
func Load(defaultConfig []byte) (*Config, error) {
	// 1. Try next to the executable
	if exe, err := os.Executable(); err == nil {
		cfgPath := filepath.Join(filepath.Dir(exe), "config.toml")
		if _, err := os.Stat(cfgPath); err == nil {
			var cfg Config
			if _, err := toml.DecodeFile(cfgPath, &cfg); err == nil {
				return &cfg, nil
			}
		}
	}

	// 2. Try current working directory (dev mode)
	if _, err := os.Stat("config.toml"); err == nil {
		var cfg Config
		if _, err := toml.DecodeFile("config.toml", &cfg); err == nil {
			return &cfg, nil
		}
	}

	// 3. Fall back to embedded default
	if len(defaultConfig) > 0 {
		var cfg Config
		if _, err := toml.NewDecoder(bytes.NewReader(defaultConfig)).Decode(&cfg); err != nil {
			return nil, err
		}
		return &cfg, nil
	}

	return nil, os.ErrNotExist
}
