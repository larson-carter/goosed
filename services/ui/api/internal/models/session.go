package models

import (
	"time"

	"github.com/google/uuid"
)

// Session tracks refresh token lifecycle state.
type Session struct {
	ID           uuid.UUID `gorm:"type:uuid;default:gen_random_uuid();primaryKey"`
	UserID       uuid.UUID `gorm:"type:uuid;not null;index"`
	RefreshToken string    `gorm:"type:text;uniqueIndex;not null"`
	ExpiresAt    time.Time `gorm:"not null"`
	CreatedAt    time.Time `gorm:"autoCreateTime"`
	RevokedAt    *time.Time

	User User `gorm:"constraint:OnDelete:CASCADE;foreignKey:UserID;references:ID"`
}
