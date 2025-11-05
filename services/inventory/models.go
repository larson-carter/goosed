package inventory

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/datatypes"
)

type factModel struct {
	ID        uuid.UUID         `gorm:"type:uuid;primaryKey"`
	MachineID uuid.UUID         `gorm:"type:uuid;not null;index"`
	Snapshot  datatypes.JSONMap `gorm:"type:jsonb"`
	CreatedAt time.Time         `gorm:"type:timestamptz;not null;default:now();autoCreateTime"`
}

func (factModel) TableName() string { return "facts" }

type auditModel struct {
	ID      int64             `gorm:"type:bigserial;primaryKey"`
	Actor   string            `gorm:"type:text;not null"`
	Action  string            `gorm:"type:text;not null"`
	Obj     string            `gorm:"type:text"`
	Details datatypes.JSONMap `gorm:"type:jsonb"`
	At      time.Time         `gorm:"type:timestamptz;not null;default:now();autoCreateTime"`
}

func (auditModel) TableName() string { return "audit" }

func mapFromJSONMap(src datatypes.JSONMap) map[string]any {
	if src == nil {
		return map[string]any{}
	}
	out := make(map[string]any, len(src))
	for k, v := range src {
		out[k] = v
	}
	return out
}

func toJSONMap(src map[string]any) datatypes.JSONMap {
	out := datatypes.JSONMap{}
	if src == nil {
		return out
	}
	for k, v := range src {
		out[k] = v
	}
	return out
}
