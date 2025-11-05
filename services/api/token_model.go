package api

import (
	"time"

	"github.com/google/uuid"
)

type tokenModel struct {
	ID        uuid.UUID `gorm:"type:uuid;primaryKey"`
	MAC       string    `gorm:"type:text;index;not null"`
	Token     string    `gorm:"type:text;uniqueIndex;not null"`
	ExpiresAt time.Time `gorm:"type:timestamptz;not null"`
	Used      bool      `gorm:"type:boolean;not null;default:false"`
	CreatedAt time.Time `gorm:"type:timestamptz;not null;default:now();autoCreateTime"`
	UpdatedAt time.Time `gorm:"type:timestamptz;not null;default:now();autoUpdateTime"`
}

func (tokenModel) TableName() string { return "tokens" }

func (m tokenModel) toToken() Token {
	return Token{
		ID:        m.ID,
		MAC:       m.MAC,
		Value:     m.Token,
		ExpiresAt: m.ExpiresAt,
		Used:      m.Used,
	}
}
