package orchestrator

import (
	"time"

	"github.com/google/uuid"
)

type runModel struct {
	ID         uuid.UUID  `gorm:"type:uuid;primaryKey"`
	MachineID  *uuid.UUID `gorm:"type:uuid"`
	Status     string     `gorm:"type:text"`
	StartedAt  *time.Time `gorm:"type:timestamptz"`
	FinishedAt *time.Time `gorm:"type:timestamptz"`
}

func (runModel) TableName() string { return "runs" }

const (
	runStatusRunning = "running"
	runStatusSuccess = "success"
)
