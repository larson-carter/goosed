package api

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/datatypes"
)

type blueprintModel struct {
	ID        uuid.UUID         `gorm:"type:uuid;primaryKey"`
	Name      string            `gorm:"type:text;not null"`
	OS        string            `gorm:"type:text;not null"`
	Version   string            `gorm:"type:text;not null"`
	Data      datatypes.JSONMap `gorm:"type:jsonb"`
	CreatedAt time.Time         `gorm:"type:timestamptz;not null;default:now();autoCreateTime"`
	UpdatedAt time.Time         `gorm:"type:timestamptz;not null;default:now();autoUpdateTime"`
}

func (blueprintModel) TableName() string { return "blueprints" }

func (m blueprintModel) toAPI() Blueprint {
	return Blueprint{
		ID:        m.ID,
		Name:      m.Name,
		OS:        m.OS,
		Version:   m.Version,
		Data:      mapFromJSONMap(m.Data),
		CreatedAt: m.CreatedAt,
		UpdatedAt: m.UpdatedAt,
	}
}
