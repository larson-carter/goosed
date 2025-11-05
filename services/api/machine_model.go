package api

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/datatypes"
)

type machineModel struct {
	ID        uuid.UUID         `gorm:"type:uuid;primaryKey"`
	MAC       string            `gorm:"type:text;uniqueIndex;not null"`
	Serial    string            `gorm:"type:text"`
	Profile   datatypes.JSONMap `gorm:"type:jsonb"`
	CreatedAt time.Time         `gorm:"type:timestamptz;not null;default:now();autoCreateTime"`
	UpdatedAt time.Time         `gorm:"type:timestamptz;not null;default:now();autoUpdateTime"`
}

func (machineModel) TableName() string { return "machines" }

func (m machineModel) toAPI() Machine {
	return Machine{
		ID:        m.ID,
		MAC:       m.MAC,
		Serial:    m.Serial,
		Profile:   mapFromJSONMap(m.Profile),
		CreatedAt: m.CreatedAt,
		UpdatedAt: m.UpdatedAt,
	}
}
