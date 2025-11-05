package inventory

import (
	"time"

	"gorm.io/datatypes"
)

type auditModel struct {
	ID      int64             `gorm:"type:bigserial;primaryKey"`
	Actor   string            `gorm:"type:text;not null"`
	Action  string            `gorm:"type:text;not null"`
	Obj     string            `gorm:"type:text"`
	Details datatypes.JSONMap `gorm:"type:jsonb"`
	At      time.Time         `gorm:"type:timestamptz;not null;default:now();autoCreateTime"`
}

func (auditModel) TableName() string { return "audit" }
