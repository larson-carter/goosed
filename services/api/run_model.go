package api

import (
	"time"

	"github.com/google/uuid"
)

type runModel struct {
	ID          uuid.UUID  `gorm:"type:uuid;primaryKey"`
	MachineID   *uuid.UUID `gorm:"type:uuid"`
	BlueprintID *uuid.UUID `gorm:"type:uuid"`
	Status      string     `gorm:"type:text"`
	StartedAt   *time.Time `gorm:"type:timestamptz"`
	FinishedAt  *time.Time `gorm:"type:timestamptz"`
	Logs        string     `gorm:"type:text"`
}

func (runModel) TableName() string { return "runs" }

func (r runModel) toAPI() Run {
	return Run{
		ID:          r.ID,
		MachineID:   valueOrZero(r.MachineID),
		BlueprintID: valueOrZero(r.BlueprintID),
		Status:      r.Status,
		StartedAt:   r.StartedAt,
		FinishedAt:  r.FinishedAt,
		Logs:        r.Logs,
	}
}

func valueOrZero(id *uuid.UUID) uuid.UUID {
	if id == nil {
		return uuid.Nil
	}
	return *id
}
