package db

import (
	"context"

	"goosed/services/ui/api/internal/models"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// Seed inserts baseline lookup data such as default roles.
func Seed(ctx context.Context, database *gorm.DB) error {
	defaultRoles := []string{"Admin", "Operator", "Viewer"}
	for _, roleName := range defaultRoles {
		role := models.Role{Name: roleName}
		if err := database.WithContext(ctx).
			Clauses(clause.OnConflict{DoNothing: true}).
			Create(&role).Error; err != nil {
			return err
		}
	}
	return nil
}
