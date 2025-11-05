package inventory

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"reflect"
	"sync"
	"time"

	"github.com/google/uuid"

	"goosed/pkg/bus"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

const (
	agentFactsSubject = "goosed.agent.facts"
	auditActor        = "agent"
	auditAction       = "facts_updated"
)

// Ingestor coordinates ingesting agent facts from NATS into the database while
// emitting audit log entries describing the changes.
type Ingestor struct {
	orm *gorm.DB
	bus *bus.Bus

	subMu sync.Mutex
	sub   io.Closer
}

// NewIngestor constructs an Ingestor for the provided dependencies.
func NewIngestor(orm *gorm.DB, bus *bus.Bus) (*Ingestor, error) {
	if orm == nil {
		return nil, errors.New("orm is required")
	}
	if bus == nil {
		return nil, errors.New("bus is required")
	}

	return &Ingestor{orm: orm, bus: bus}, nil
}

// Start subscribes to agent fact updates and processes them until ctx is cancelled.
func (i *Ingestor) Start(ctx context.Context) error {
	if i == nil {
		return errors.New("nil ingestor")
	}
	if ctx == nil {
		return errors.New("context is required")
	}

	handler := func(msgCtx context.Context, data []byte) error {
		return i.handleFact(msgCtx, data)
	}

	sub, err := i.bus.Subscribe(ctx, agentFactsSubject, "inventory-facts", handler)
	if err != nil {
		return err
	}

	i.subMu.Lock()
	i.sub = sub
	i.subMu.Unlock()

	return nil
}

// Close stops the underlying subscription if it was created.
func (i *Ingestor) Close() error {
	if i == nil {
		return nil
	}

	i.subMu.Lock()
	defer i.subMu.Unlock()

	if i.sub == nil {
		return nil
	}
	err := i.sub.Close()
	i.sub = nil
	return err
}

func (i *Ingestor) handleFact(ctx context.Context, data []byte) error {
	var evt factEvent
	if err := json.Unmarshal(data, &evt); err != nil {
		return err
	}
	if evt.FactID == uuid.Nil {
		return errors.New("fact_id missing from event")
	}
	if evt.MachineID == uuid.Nil {
		return errors.New("machine_id missing from event")
	}
	if evt.Snapshot == nil {
		evt.Snapshot = map[string]any{}
	}
	if evt.CreatedAt.IsZero() {
		evt.CreatedAt = time.Now().UTC()
	}

	previous, err := i.previousSnapshot(ctx, evt.MachineID, evt.FactID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			previous = map[string]any{}
		} else {
			return err
		}
	}

	if err := i.insertFact(ctx, evt); err != nil {
		return err
	}

	diff := computeDiff(previous, evt.Snapshot)

	return i.insertAudit(ctx, evt, diff)
}

func (i *Ingestor) previousSnapshot(ctx context.Context, machineID, currentFactID uuid.UUID) (map[string]any, error) {
	var previous factModel
	err := i.orm.WithContext(ctx).
		Where("machine_id = ? AND id <> ?", machineID, currentFactID).
		Order("created_at DESC").
		First(&previous).Error
	if err != nil {
		return nil, err
	}
	return mapFromJSONMap(previous.Snapshot), nil
}

func (i *Ingestor) insertFact(ctx context.Context, evt factEvent) error {
	snapshot := toJSONMap(evt.Snapshot)

	fact := factModel{
		ID:        evt.FactID,
		MachineID: evt.MachineID,
		Snapshot:  snapshot,
		CreatedAt: evt.CreatedAt,
	}

	return i.orm.WithContext(ctx).
		Clauses(clause.OnConflict{DoNothing: true}).
		Create(&fact).Error
}

func (i *Ingestor) insertAudit(ctx context.Context, evt factEvent, diff map[string]map[string]any) error {
	details := map[string]any{
		"machine_id": evt.MachineID.String(),
		"fact_id":    evt.FactID.String(),
		"changes":    diff,
	}

	audit := auditModel{
		Actor:   auditActor,
		Action:  auditAction,
		Obj:     evt.MachineID.String(),
		Details: toJSONMap(details),
	}

	return i.orm.WithContext(ctx).Create(&audit).Error
}

func computeDiff(previous, current map[string]any) map[string]map[string]any {
	if previous == nil {
		previous = map[string]any{}
	}
	if current == nil {
		current = map[string]any{}
	}

	diff := make(map[string]map[string]any)

	for key, prevVal := range previous {
		curVal, ok := current[key]
		if !ok {
			diff[key] = map[string]any{"old": prevVal, "new": nil}
			continue
		}
		if !reflect.DeepEqual(prevVal, curVal) {
			diff[key] = map[string]any{"old": prevVal, "new": curVal}
		}
	}

	for key, curVal := range current {
		if _, seen := previous[key]; seen {
			continue
		}
		diff[key] = map[string]any{"old": nil, "new": curVal}
	}

	return diff
}
