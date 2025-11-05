package orchestrator

import "github.com/google/uuid"

type machineEnrolledEvent struct {
	MachineID uuid.UUID `json:"machine_id"`
}

type agentFactsEvent struct {
	MachineID uuid.UUID      `json:"machine_id"`
	Snapshot  map[string]any `json:"snapshot"`
}

type runLifecycleEvent struct {
	RunID     uuid.UUID `json:"run_id"`
	MachineID uuid.UUID `json:"machine_id"`
	Status    string    `json:"status"`
}
