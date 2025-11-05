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
