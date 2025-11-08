package models

import (
	"time"

	"github.com/google/uuid"
)

// ResetToken stores password reset tokens for users.
type ResetToken struct {
	ID         uuid.UUID `gorm:"type:uuid;default:gen_random_uuid();primaryKey"`
	UserID     uuid.UUID `gorm:"type:uuid;not null;index"`
	Token      string    `gorm:"type:text;uniqueIndex;not null"`
	ExpiresAt  time.Time `gorm:"not null"`
	CreatedAt  time.Time `gorm:"autoCreateTime"`
	ConsumedAt *time.Time

	User User `gorm:"constraint:OnDelete:CASCADE;foreignKey:UserID;references:ID"`
}
