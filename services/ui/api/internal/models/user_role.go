package models

import (
	"time"

	"github.com/google/uuid"
)

// UserRole ties a user to a role with assignment metadata.
type UserRole struct {
	UserID    uuid.UUID `gorm:"type:uuid;primaryKey"`
	RoleID    uint      `gorm:"primaryKey"`
	CreatedAt time.Time `gorm:"autoCreateTime"`

	User User `gorm:"constraint:OnDelete:CASCADE;foreignKey:UserID;references:ID"`
	Role Role `gorm:"constraint:OnDelete:CASCADE;foreignKey:RoleID;references:ID"`
}
