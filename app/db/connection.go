package db

import (
	"ricemill/app/config"
	"fmt"
	"time"

	gomysql "github.com/go-sql-driver/mysql"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

var DB *gorm.DB

// Connect opens a MySQL connection using the DSN from config.
func Connect(cfg *config.Config) error {
	gormCfg := &gorm.Config{
		Logger:                                   logger.Default.LogMode(logger.Warn),
		DisableForeignKeyConstraintWhenMigrating: true,
	}

	// Parse DSN so we can inject sql_mode as a session variable on every
	// new connection in the pool.  Wrapping the value in single quotes
	// makes the driver emit:  SET sql_mode='...'  which is valid MySQL.
	// This removes NO_ZERO_DATE / NO_ZERO_IN_DATE so GORM's datetime(3)
	// default values don't trigger Error 1067.
	dsnCfg, err := gomysql.ParseDSN(cfg.Database.DSN)
	if err != nil {
		return fmt.Errorf("invalid DSN: %w", err)
	}
	if dsnCfg.Params == nil {
		dsnCfg.Params = make(map[string]string)
	}
	dsnCfg.Params["sql_mode"] = "'ONLY_FULL_GROUP_BY,STRICT_TRANS_TABLES,ERROR_FOR_DIVISION_BY_ZERO,NO_ENGINE_SUBSTITUTION'"

	db, err := gorm.Open(mysql.Open(dsnCfg.FormatDSN()), gormCfg)
	if err != nil {
		return err
	}

	sqlDB, err := db.DB()
	if err != nil {
		return err
	}

	maxOpen := cfg.Database.MaxOpenConns
	if maxOpen <= 0 {
		maxOpen = 10
	}
	maxIdle := cfg.Database.MaxIdleConns
	if maxIdle <= 0 {
		maxIdle = 5
	}

	sqlDB.SetMaxOpenConns(maxOpen)
	sqlDB.SetMaxIdleConns(maxIdle)
	sqlDB.SetConnMaxLifetime(time.Hour)

	DB = db
	return nil
}

// Ping verifies the database connection is alive.
func Ping() error {
	sqlDB, err := DB.DB()
	if err != nil {
		return err
	}
	return sqlDB.Ping()
}
