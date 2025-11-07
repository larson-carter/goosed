package db

import (
	"context"
	"time"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
	"gorm.io/gorm/schema"
)

// Connect establishes a PostgreSQL backed GORM session configured for the UI auth schema.
func Connect(ctx context.Context, dsn string) (*gorm.DB, error) {
	gormCfg := &gorm.Config{
		Logger: logger.Default.LogMode(logger.Warn),
		NamingStrategy: schema.NamingStrategy{
			TablePrefix:   "ui_auth.",
			SingularTable: false,
		},
	}

	database, err := gorm.Open(postgres.Open(dsn), gormCfg)
	if err != nil {
		return nil, err
	}

	sqlDB, err := database.DB()
	if err != nil {
		return nil, err
	}

	sqlDB.SetConnMaxLifetime(30 * time.Minute)
	sqlDB.SetConnMaxIdleTime(5 * time.Minute)
	sqlDB.SetMaxOpenConns(25)
	sqlDB.SetMaxIdleConns(5)

	if err := database.WithContext(ctx).Exec(`CREATE SCHEMA IF NOT EXISTS ui_auth`).Error; err != nil {
		return nil, err
	}

	return database, nil
}

// Close releases the underlying sql.DB resources for the provided GORM handle.
func Close(database *gorm.DB) error {
	sqlDB, err := database.DB()
	if err != nil {
		return err
	}
	return sqlDB.Close()
}
