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
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"goosed/pkg/bus"
	"goosed/pkg/db"
)

const (
	agentFactsSubject = "goosed.agent.facts"
	auditActor        = "agent"
	auditAction       = "facts_updated"
)

type factEvent struct {
	FactID    uuid.UUID      `json:"fact_id"`
	MachineID uuid.UUID      `json:"machine_id"`
	Snapshot  map[string]any `json:"snapshot"`
	CreatedAt time.Time      `json:"created_at"`
}

// Ingestor coordinates ingesting agent facts from NATS into the database while
// emitting audit log entries describing the changes.
type Ingestor struct {
	pool *pgxpool.Pool
	bus  *bus.Bus

	subMu sync.Mutex
	sub   io.Closer
}

// NewIngestor constructs an Ingestor for the provided dependencies.
func NewIngestor(pool *pgxpool.Pool, bus *bus.Bus) (*Ingestor, error) {
	if pool == nil {
		return nil, errors.New("database pool is required")
	}
	if bus == nil {
		return nil, errors.New("bus is required")
	}

	return &Ingestor{pool: pool, bus: bus}, nil
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
	if err != nil && !errors.Is(err, pgx.ErrNoRows) {
		return err
	}

	if err := i.insertFact(ctx, evt); err != nil {
		return err
	}

	diff := computeDiff(previous, evt.Snapshot)

	return i.insertAudit(ctx, evt, diff)
}

func (i *Ingestor) previousSnapshot(ctx context.Context, machineID, currentFactID uuid.UUID) (map[string]any, error) {
	var raw []byte
	err := db.Get(ctx, i.pool, &raw, `
SELECT snapshot
FROM facts
WHERE machine_id = $1 AND id <> $2
ORDER BY created_at DESC
LIMIT 1
`, machineID, currentFactID)
	if err != nil {
		return nil, err
	}
	if len(raw) == 0 {
		return map[string]any{}, nil
	}

	var snapshot map[string]any
	if err := json.Unmarshal(raw, &snapshot); err != nil {
		return nil, err
	}
	return snapshot, nil
}

func (i *Ingestor) insertFact(ctx context.Context, evt factEvent) error {
	snapshotBytes, err := json.Marshal(evt.Snapshot)
	if err != nil {
		return err
	}

	_, err = db.Exec(ctx, i.pool, `
INSERT INTO facts (id, machine_id, snapshot, created_at)
VALUES ($1, $2, $3::jsonb, $4)
ON CONFLICT (id) DO NOTHING
`, evt.FactID, evt.MachineID, snapshotBytes, evt.CreatedAt)
	return err
}

func (i *Ingestor) insertAudit(ctx context.Context, evt factEvent, diff map[string]map[string]any) error {
	details := map[string]any{
		"machine_id": evt.MachineID.String(),
		"fact_id":    evt.FactID.String(),
		"changes":    diff,
	}

	detailsBytes, err := json.Marshal(details)
	if err != nil {
		return err
	}

	_, err = db.Exec(ctx, i.pool, `
INSERT INTO audit (actor, action, obj, details)
VALUES ($1, $2, $3, $4::jsonb)
`, auditActor, auditAction, evt.MachineID.String(), detailsBytes)
	return err
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
