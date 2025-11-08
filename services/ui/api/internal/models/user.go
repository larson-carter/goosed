package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// User represents an end user of the goose'd UI platform.
type User struct {
	ID           uuid.UUID      `gorm:"type:uuid;default:gen_random_uuid();primaryKey"`
	Email        string         `gorm:"type:text;uniqueIndex;not null"`
	Name         string         `gorm:"type:text;not null"`
	PasswordHash string         `gorm:"type:text;not null"`
	IsVerified   bool           `gorm:"not null;default:false"`
	CreatedAt    time.Time      `gorm:"autoCreateTime"`
	UpdatedAt    time.Time      `gorm:"autoUpdateTime"`
	DeletedAt    gorm.DeletedAt `gorm:"index"`

	Sessions    []Session    `gorm:"constraint:OnDelete:CASCADE"`
	EmailTokens []EmailToken `gorm:"constraint:OnDelete:CASCADE"`
	ResetTokens []ResetToken `gorm:"constraint:OnDelete:CASCADE"`
	Roles       []Role       `gorm:"many2many:user_roles"`
	Invites     []Invite     `gorm:"foreignKey:InviterID;constraint:OnDelete:SET NULL"`
	AuditLogs   []AuditLog   `gorm:"foreignKey:ActorID"`
}
