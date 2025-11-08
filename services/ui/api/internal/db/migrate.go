package db

import (
	"context"

	"goosed/services/ui/api/internal/models"
	"gorm.io/gorm"
)

// Migrate performs schema migrations for the persistent models.
func Migrate(ctx context.Context, database *gorm.DB) error {
	if err := database.SetupJoinTable(&models.User{}, "Roles", &models.UserRole{}); err != nil {
		return err
	}

	return database.WithContext(ctx).AutoMigrate(
		&models.User{},
		&models.Role{},
		&models.UserRole{},
		&models.Session{},
		&models.EmailToken{},
		&models.ResetToken{},
		&models.Invite{},
		&models.AuditLog{},
	)
}
