package api

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/datatypes"
)

type factModel struct {
	ID        uuid.UUID         `gorm:"type:uuid;primaryKey"`
	MachineID uuid.UUID         `gorm:"type:uuid;not null"`
	Snapshot  datatypes.JSONMap `gorm:"type:jsonb"`
	CreatedAt time.Time         `gorm:"type:timestamptz;not null;default:now();autoCreateTime"`
}

func (factModel) TableName() string { return "facts" }

func (f factModel) toMap() map[string]any {
	return map[string]any{
		"id":         f.ID,
		"machine_id": f.MachineID,
		"snapshot":   mapFromJSONMap(f.Snapshot),
		"created_at": f.CreatedAt,
	}
}
