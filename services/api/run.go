package api

import (
	"time"

	"github.com/google/uuid"
)

// Run represents a provisioning workflow execution associated with a machine and blueprint.
type Run struct {
	ID          uuid.UUID  `json:"id" db:"id"`
	MachineID   uuid.UUID  `json:"machine_id" db:"machine_id"`
	BlueprintID uuid.UUID  `json:"blueprint_id" db:"blueprint_id"`
	Status      string     `json:"status" db:"status"`
	StartedAt   *time.Time `json:"started_at" db:"started_at"`
	FinishedAt  *time.Time `json:"finished_at" db:"finished_at"`
	Logs        string     `json:"logs" db:"logs"`
}
