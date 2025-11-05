package orchestrator

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"

	"goosed/pkg/bus"
	"gorm.io/gorm"
)

const (
	machinesEnrolledSubject = "goosed.machines.enrolled"
	agentFactsSubject       = "goosed.agent.facts"
	runStartedSubject       = "goosed.runs.started"
	runFinishedSubject      = "goosed.runs.finished"
)

// StateMachine coordinates provisioning runs in response to machine lifecycle
// and agent fact events.
type StateMachine struct {
	orm *gorm.DB
	bus *bus.Bus

	activeMu   sync.RWMutex
	activeRuns map[uuid.UUID]uuid.UUID

	subsMu sync.Mutex
	subs   []io.Closer
}

// NewStateMachine creates a state machine bound to the provided dependencies.
func NewStateMachine(orm *gorm.DB, bus *bus.Bus) (*StateMachine, error) {
	if orm == nil {
		return nil, errors.New("orm is required")
	}
	if bus == nil {
		return nil, errors.New("bus is required")
	}

	return &StateMachine{
		orm:        orm,
		bus:        bus,
		activeRuns: make(map[uuid.UUID]uuid.UUID),
	}, nil
}

// Start registers NATS subscriptions and begins processing events.
func (sm *StateMachine) Start(ctx context.Context) error {
	if sm == nil {
		return errors.New("nil state machine")
	}
	if ctx == nil {
		return errors.New("context is required")
	}

	specs := []struct {
		subject string
		durable string
		handler func(context.Context, []byte) error
	}{
		{machinesEnrolledSubject, "orchestrator-machines", sm.handleMachineEnrolled},
		{agentFactsSubject, "orchestrator-facts", sm.handleAgentFacts},
		{runStartedSubject, "orchestrator-runs-started", sm.handleRunStarted},
		{runFinishedSubject, "orchestrator-runs-finished", sm.handleRunFinished},
	}

	for _, spec := range specs {
		closer, err := sm.bus.Subscribe(ctx, spec.subject, spec.durable, spec.handler)
		if err != nil {
			sm.Close()
			return err
		}
		sm.subsMu.Lock()
		sm.subs = append(sm.subs, closer)
		sm.subsMu.Unlock()
	}

	return nil
}

// Close tears down active subscriptions.
func (sm *StateMachine) Close() error {
	if sm == nil {
		return nil
	}

	sm.subsMu.Lock()
	defer sm.subsMu.Unlock()

	var firstErr error
	for _, sub := range sm.subs {
		if sub == nil {
			continue
		}
		if err := sub.Close(); err != nil && firstErr == nil {
			firstErr = err
		}
	}
	sm.subs = nil
	return firstErr
}

func (sm *StateMachine) handleMachineEnrolled(ctx context.Context, data []byte) error {
	var evt machineEnrolledEvent
	if err := json.Unmarshal(data, &evt); err != nil {
		return err
	}
	if evt.MachineID == uuid.Nil {
		return errors.New("machine_id missing from enrollment event")
	}

	if runID, ok := sm.getActiveRun(evt.MachineID); ok && runID != uuid.Nil {
		return nil
	}

	var existing runModel
	err := sm.orm.WithContext(ctx).
		Where("machine_id = ? AND status = ?", evt.MachineID, runStatusRunning).
		Order("started_at DESC").
		First(&existing).Error
	switch {
	case err == nil:
		sm.setActiveRun(evt.MachineID, existing.ID)
		return nil
	case !errors.Is(err, gorm.ErrRecordNotFound):
		return err
	}

	runID := uuid.New()
	startedAt := time.Now().UTC()
	machineID := evt.MachineID
	run := runModel{
		ID:        runID,
		MachineID: &machineID,
		Status:    runStatusRunning,
		StartedAt: &startedAt,
	}
	if err := sm.orm.WithContext(ctx).Create(&run).Error; err != nil {
		return err
	}

	sm.setActiveRun(evt.MachineID, runID)

	payload := map[string]any{
		"run_id":     runID,
		"machine_id": evt.MachineID,
		"status":     runStatusRunning,
		"started_at": startedAt,
	}

	return sm.bus.Publish(ctx, runStartedSubject, payload)
}

func (sm *StateMachine) handleAgentFacts(ctx context.Context, data []byte) error {
	var evt agentFactsEvent
	if err := json.Unmarshal(data, &evt); err != nil {
		return err
	}
	if evt.MachineID == uuid.Nil {
		return errors.New("machine_id missing from facts event")
	}
	if evt.Snapshot == nil {
		evt.Snapshot = map[string]any{}
	}

	if !isPostInstallDone(evt.Snapshot) {
		return nil
	}

	runID, ok := sm.getActiveRun(evt.MachineID)
	if !ok {
		var run runModel
		err := sm.orm.WithContext(ctx).
			Where("machine_id = ? AND status = ?", evt.MachineID, runStatusRunning).
			Order("started_at DESC").
			First(&run).Error
		if err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return nil
			}
			return err
		}
		runID = run.ID
		sm.setActiveRun(evt.MachineID, runID)
	}
	if runID == uuid.Nil {
		return nil
	}

	finishedAt := time.Now().UTC()
	updates := map[string]any{
		"status":      runStatusSuccess,
		"finished_at": finishedAt,
	}
	if err := sm.orm.WithContext(ctx).
		Model(&runModel{}).
		Where("id = ?", runID).
		Updates(updates).Error; err != nil {
		return err
	}

	sm.clearActiveRun(evt.MachineID, runID)

	payload := map[string]any{
		"run_id":      runID,
		"machine_id":  evt.MachineID,
		"status":      runStatusSuccess,
		"finished_at": finishedAt,
	}

	return sm.bus.Publish(ctx, runFinishedSubject, payload)
}

func (sm *StateMachine) handleRunStarted(ctx context.Context, data []byte) error {
	var evt runLifecycleEvent
	if err := json.Unmarshal(data, &evt); err != nil {
		return err
	}
	if evt.MachineID == uuid.Nil || evt.RunID == uuid.Nil {
		return nil
	}
	if strings.EqualFold(evt.Status, runStatusRunning) {
		sm.setActiveRun(evt.MachineID, evt.RunID)
	}
	return nil
}

func (sm *StateMachine) handleRunFinished(ctx context.Context, data []byte) error {
	var evt runLifecycleEvent
	if err := json.Unmarshal(data, &evt); err != nil {
		return err
	}
	if evt.MachineID == uuid.Nil || evt.RunID == uuid.Nil {
		return nil
	}
	sm.clearActiveRun(evt.MachineID, evt.RunID)
	return nil
}

func (sm *StateMachine) setActiveRun(machineID, runID uuid.UUID) {
	sm.activeMu.Lock()
	defer sm.activeMu.Unlock()
	if sm.activeRuns == nil {
		sm.activeRuns = make(map[uuid.UUID]uuid.UUID)
	}
	sm.activeRuns[machineID] = runID
}

func (sm *StateMachine) clearActiveRun(machineID, runID uuid.UUID) {
	sm.activeMu.Lock()
	defer sm.activeMu.Unlock()
	if current, ok := sm.activeRuns[machineID]; ok && current == runID {
		delete(sm.activeRuns, machineID)
	}
}

func (sm *StateMachine) getActiveRun(machineID uuid.UUID) (uuid.UUID, bool) {
	sm.activeMu.RLock()
	defer sm.activeMu.RUnlock()
	runID, ok := sm.activeRuns[machineID]
	return runID, ok
}

func isPostInstallDone(snapshot map[string]any) bool {
	if snapshot == nil {
		return false
	}
	value, ok := snapshot["postinstall_done"]
	if !ok {
		return false
	}
	switch v := value.(type) {
	case bool:
		return v
	case string:
		return strings.EqualFold(v, "true")
	case float64:
		return v != 0
	default:
		return false
	}
}
