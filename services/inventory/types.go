package inventory

import (
	"time"

	"github.com/google/uuid"
)

type factEvent struct {
	FactID    uuid.UUID      `json:"fact_id"`
	MachineID uuid.UUID      `json:"machine_id"`
	Snapshot  map[string]any `json:"snapshot"`
	CreatedAt time.Time      `json:"created_at"`
}
