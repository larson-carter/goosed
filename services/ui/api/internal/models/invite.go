package models

import (
	"time"

	"github.com/google/uuid"
)

// Invite captures pending invitations sent to prospective users.
type Invite struct {
	ID         uuid.UUID  `gorm:"type:uuid;default:gen_random_uuid();primaryKey"`
	Email      string     `gorm:"type:text;not null"`
	InviterID  *uuid.UUID `gorm:"type:uuid;index"`
	Token      string     `gorm:"type:text;uniqueIndex;not null"`
	ExpiresAt  time.Time  `gorm:"not null"`
	CreatedAt  time.Time  `gorm:"autoCreateTime"`
	AcceptedAt *time.Time

	Inviter *User `gorm:"constraint:OnDelete:SET NULL;foreignKey:InviterID;references:ID"`
}
