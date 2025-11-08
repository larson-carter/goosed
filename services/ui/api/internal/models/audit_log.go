package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/datatypes"
)

// AuditLog captures notable authentication and administrative events.
type AuditLog struct {
	ID         uuid.UUID      `gorm:"type:uuid;default:gen_random_uuid();primaryKey"`
	ActorID    *uuid.UUID     `gorm:"type:uuid;index"`
	Action     string         `gorm:"type:text;not null"`
	TargetType string         `gorm:"type:text;not null"`
	TargetID   *string        `gorm:"type:text"`
	Metadata   datatypes.JSON `gorm:"type:jsonb;default:'{}'::jsonb"`
	CreatedAt  time.Time      `gorm:"autoCreateTime"`

	Actor *User `gorm:"constraint:OnUpdate:CASCADE,OnDelete:SET NULL;foreignKey:ActorID;references:ID"`
}
