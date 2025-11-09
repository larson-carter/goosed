package api

import (
	"time"

	"github.com/google/uuid"
)

type machineListItem struct {
	Machine    Machine          `json:"machine"`
	Status     string           `json:"status"`
	LatestFact *machineListFact `json:"latest_fact,omitempty"`
	RecentRuns []Run            `json:"recent_runs,omitempty"`
}

type machineListFact struct {
	ID        uuid.UUID      `json:"id"`
	Snapshot  map[string]any `json:"snapshot"`
	CreatedAt time.Time      `json:"created_at"`
}

const (
	machineStatusReady        = "ready"
	machineStatusProvisioning = "provisioning"
	machineStatusError        = "error"
	machineStatusOffline      = "offline"
)
