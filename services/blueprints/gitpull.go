package blueprints

import (
	"context"
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"reflect"
	"sync"
	"time"

	"github.com/google/uuid"

	"goosed/pkg/bus"
)

const (
	defaultInfraPath    = "./infra"
	blueprintsDir       = "blueprints"
	workflowsDir        = "workflows"
	blueprintsTopicName = "goosed.blueprints.updated"
)

// Snapshot represents the in-memory view of blueprints and workflows loaded from disk.
type Snapshot struct {
	Version    string
	UpdatedAt  time.Time
	Blueprints map[string]string
	Workflows  map[string]string
}

// Watcher monitors the local infrastructure repository and publishes updates when
// changes are detected.
type Watcher struct {
	bus       *bus.Bus
	infraPath string
	interval  time.Duration

	mu       sync.RWMutex
	snapshot Snapshot
}

// NewWatcher builds a new Watcher instance. If infraPath is empty the INFRA_PATH
// environment variable is considered before falling back to ./infra. Interval
// defaults to 30 seconds when not provided.
func NewWatcher(b *bus.Bus, infraPath string, interval time.Duration) *Watcher {
	if infraPath == "" {
		infraPath = os.Getenv("INFRA_PATH")
	}
	if infraPath == "" {
		infraPath = defaultInfraPath
	}
	if interval <= 0 {
		interval = 30 * time.Second
	}

	return &Watcher{
		bus:       b,
		infraPath: infraPath,
		interval:  interval,
		snapshot: Snapshot{
			Blueprints: map[string]string{},
			Workflows:  map[string]string{},
		},
	}
}

// Start begins polling the infra repository and publishing updates. It blocks
// until the provided context is cancelled or an unrecoverable error occurs.
func (w *Watcher) Start(ctx context.Context) error {
	if w == nil {
		return errors.New("nil watcher")
	}
	if w.bus == nil {
		return errors.New("bus is required")
	}
	if ctx == nil {
		return errors.New("context is required")
	}

	// Perform an initial sync so consumers receive the latest view right away.
	if err := w.sync(ctx, true); err != nil {
		return err
	}

	ticker := time.NewTicker(w.interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
			if err := w.sync(ctx, false); err != nil {
				return err
			}
		}
	}
}

// Snapshot returns a copy of the latest cached state.
func (w *Watcher) Snapshot() Snapshot {
	if w == nil {
		return Snapshot{}
	}

	w.mu.RLock()
	defer w.mu.RUnlock()

	return Snapshot{
		Version:    w.snapshot.Version,
		UpdatedAt:  w.snapshot.UpdatedAt,
		Blueprints: cloneMap(w.snapshot.Blueprints),
		Workflows:  cloneMap(w.snapshot.Workflows),
	}
}

func (w *Watcher) sync(ctx context.Context, force bool) error {
	current, err := w.readSnapshot()
	if err != nil {
		return err
	}

	w.mu.Lock()
	changed := !reflect.DeepEqual(w.snapshot.Blueprints, current.Blueprints) || !reflect.DeepEqual(w.snapshot.Workflows, current.Workflows)
	if w.snapshot.Version == "" {
		changed = true
	}
	if changed {
		current.Version = uuid.NewString()
		current.UpdatedAt = time.Now().UTC()
		w.snapshot = current
	} else {
		current = w.snapshot
	}
	w.mu.Unlock()

	if !changed && !force {
		return nil
	}

	payload := map[string]any{
		"version":          current.Version,
		"updated_at":       current.UpdatedAt,
		"blueprints_count": len(current.Blueprints),
		"workflows_count":  len(current.Workflows),
	}

	return w.bus.Publish(ctx, blueprintsTopicName, payload)
}

func (w *Watcher) readSnapshot() (Snapshot, error) {
	base := w.infraPath
	blueprintsPath := filepath.Join(base, blueprintsDir)
	workflowsPath := filepath.Join(base, workflowsDir)

	blueprints, err := readFiles(blueprintsPath)
	if err != nil {
		return Snapshot{}, err
	}
	workflows, err := readFiles(workflowsPath)
	if err != nil {
		return Snapshot{}, err
	}

	return Snapshot{
		Blueprints: blueprints,
		Workflows:  workflows,
	}, nil
}

func readFiles(root string) (map[string]string, error) {
	result := map[string]string{}

	info, err := os.Stat(root)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return result, nil
		}
		return nil, err
	}
	if !info.IsDir() {
		return nil, errors.New("path is not a directory")
	}

	err = filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(root, path)
		if err != nil {
			return err
		}
		result[filepath.ToSlash(rel)] = string(data)
		return nil
	})
	if err != nil {
		return nil, err
	}

	return result, nil
}

func cloneMap(src map[string]string) map[string]string {
	if src == nil {
		return map[string]string{}
	}
	out := make(map[string]string, len(src))
	for k, v := range src {
		out[k] = v
	}
	return out
}
