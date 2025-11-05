package api

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"goosed/pkg/render"
)

const (
	defaultTokenTTL  = 5 * time.Minute
	presignURLExpiry = 15 * time.Minute
	machinesTopic    = "goosed.machines.enrolled"
	agentFactsTopic  = "goosed.agent.facts"
	runStartedTopic  = "goosed.runs.started"
	runFinishedTopic = "goosed.runs.finished"
)

// Config controls runtime behaviour for the API handlers.
type Config struct {
	APIBase        string
	TokenTTL       time.Duration
	ArtifactBucket string
}

// API wires dependencies, template renderer, and configuration for HTTP handlers.
type API struct {
	store    *Store
	renderer *render.Engine
	config   Config
	tokens   *tokenStore
}

// New initialises the API layer with sane defaults applied to the provided configuration.
func New(store *Store, renderer *render.Engine, cfg Config) (*API, error) {
	if store == nil {
		return nil, errors.New("store is required")
	}
	if store.DB == nil {
		return nil, errors.New("store DB is required")
	}
	if renderer == nil {
		return nil, errors.New("renderer is required")
	}

	if cfg.TokenTTL <= 0 {
		cfg.TokenTTL = defaultTokenTTL
	}
	if cfg.ArtifactBucket == "" {
		cfg.ArtifactBucket = os.Getenv("S3_BUCKET")
	}
	if cfg.ArtifactBucket == "" {
		return nil, errors.New("artifact bucket is required")
	}

	return &API{
		store:    store,
		renderer: renderer,
		config:   cfg,
		tokens:   newTokenStore(cfg.TokenTTL),
	}, nil
}

// Routes constructs the chi router containing all API endpoints.
func (a *API) Routes() (http.Handler, error) {
	if a == nil {
		return nil, errors.New("nil api")
	}

	r := chi.NewRouter()
	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(middleware.Recoverer)
	r.Use(middleware.Timeout(60 * time.Second))

	r.Route("/v1", func(r chi.Router) {
		r.Post("/machines", a.handleUpsertMachine)
		r.Get("/boot/ipxe", a.handleIPXE)
		r.Get("/render/kickstart", a.handleKickstart)
		r.Get("/render/unattend", a.handleUnattend)
		r.Post("/artifacts", a.handleArtifacts)
		r.Post("/agents/facts", a.handleFacts)
		r.Post("/runs/start", a.handleRunStart)
		r.Post("/runs/finish", a.handleRunFinish)
	})

	return r, nil
}

type tokenEntry struct {
	MAC     string
	Expires time.Time
}

type tokenStore struct {
	ttl    time.Duration
	mu     sync.Mutex
	tokens map[string]tokenEntry
}

func newTokenStore(ttl time.Duration) *tokenStore {
	if ttl <= 0 {
		ttl = defaultTokenTTL
	}
	return &tokenStore{
		ttl:    ttl,
		tokens: make(map[string]tokenEntry),
	}
}

func (ts *tokenStore) issue(mac string) string {
	ts.mu.Lock()
	defer ts.mu.Unlock()

	for key, entry := range ts.tokens {
		if time.Now().After(entry.Expires) {
			delete(ts.tokens, key)
		}
	}

	token := uuid.New().String()
	ts.tokens[token] = tokenEntry{MAC: mac, Expires: time.Now().Add(ts.ttl)}
	return token
}

func (a *API) handleUpsertMachine(w http.ResponseWriter, r *http.Request) {
	var req struct {
		MAC     string         `json:"mac"`
		Serial  string         `json:"serial"`
		Profile map[string]any `json:"profile"`
	}
	if err := decodeJSON(r, &req); err != nil {
		respondError(w, http.StatusBadRequest, err)
		return
	}

	req.MAC = strings.TrimSpace(req.MAC)
	if req.MAC == "" {
		respondError(w, http.StatusBadRequest, errors.New("mac is required"))
		return
	}
	req.MAC = strings.ToLower(req.MAC)
	if req.Profile == nil {
		req.Profile = map[string]any{}
	}

	now := time.Now().UTC()
	id := uuid.New()

	payload, err := json.Marshal(req.Profile)
	if err != nil {
		respondError(w, http.StatusBadRequest, fmt.Errorf("marshal profile: %w", err))
		return
	}

	query := `
        INSERT INTO machines (id, mac, serial, profile, created_at, updated_at)
        VALUES ($1, $2, $3, $4::jsonb, $5, $5)
        ON CONFLICT (mac) DO UPDATE SET
            serial = EXCLUDED.serial,
            profile = EXCLUDED.profile,
            updated_at = EXCLUDED.updated_at
        RETURNING id, mac, serial, profile, created_at, updated_at;
    `

	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	row := a.store.DB.QueryRow(ctx, query, id, req.MAC, req.Serial, string(payload), now)
	var m Machine
	var profileRaw []byte
	if err := row.Scan(&m.ID, &m.MAC, &m.Serial, &profileRaw, &m.CreatedAt, &m.UpdatedAt); err != nil {
		respondError(w, http.StatusInternalServerError, err)
		return
	}
	if len(profileRaw) > 0 {
		if err := json.Unmarshal(profileRaw, &m.Profile); err != nil {
			respondError(w, http.StatusInternalServerError, fmt.Errorf("decode profile: %w", err))
			return
		}
	} else {
		m.Profile = map[string]any{}
	}

	a.publishJSON(machinesTopic, map[string]any{
		"machine_id": m.ID,
		"mac":        m.MAC,
	})

	respondJSON(w, http.StatusOK, map[string]any{"machine": m})
}

func (a *API) handleIPXE(w http.ResponseWriter, r *http.Request) {
	mac := strings.TrimSpace(r.URL.Query().Get("mac"))
	if mac == "" {
		respondError(w, http.StatusBadRequest, errors.New("mac query parameter is required"))
		return
	}
	mac = strings.ToLower(mac)

	machine, err := a.fetchMachineByMAC(r.Context(), mac)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			respondError(w, http.StatusNotFound, fmt.Errorf("machine with mac %s not found", mac))
			return
		}
		respondError(w, http.StatusInternalServerError, err)
		return
	}

	token := a.tokens.issue(machine.MAC)

	apiBase := a.config.APIBase
	if apiBase == "" {
		scheme := "http"
		if r.TLS != nil {
			scheme = "https"
		}
		apiBase = fmt.Sprintf("%s://%s", scheme, r.Host)
	}

	payload := map[string]any{
		"Token":   token,
		"MAC":     machine.MAC,
		"APIBase": apiBase,
	}

	rendered, err := a.renderer.Render("ipxe.tmpl", payload)
	if err != nil {
		respondError(w, http.StatusInternalServerError, err)
		return
	}

	w.Header().Set("Content-Type", "text/plain")
	_, _ = w.Write([]byte(rendered))
}

func (a *API) handleKickstart(w http.ResponseWriter, r *http.Request) {
	machineID, err := uuid.Parse(strings.TrimSpace(r.URL.Query().Get("machine_id")))
	if err != nil {
		respondError(w, http.StatusBadRequest, errors.New("valid machine_id is required"))
		return
	}

	machine, err := a.fetchMachineByID(r.Context(), machineID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			respondError(w, http.StatusNotFound, fmt.Errorf("machine %s not found", machineID))
			return
		}
		respondError(w, http.StatusInternalServerError, err)
		return
	}

	rendered, err := a.renderer.Render("kickstart.tmpl", map[string]any{
		"Machine": machine,
		"Profile": machine.Profile,
	})
	if err != nil {
		respondError(w, http.StatusInternalServerError, err)
		return
	}

	w.Header().Set("Content-Type", "text/plain")
	_, _ = w.Write([]byte(rendered))
}

func (a *API) handleUnattend(w http.ResponseWriter, r *http.Request) {
	machineID, err := uuid.Parse(strings.TrimSpace(r.URL.Query().Get("machine_id")))
	if err != nil {
		respondError(w, http.StatusBadRequest, errors.New("valid machine_id is required"))
		return
	}

	machine, err := a.fetchMachineByID(r.Context(), machineID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			respondError(w, http.StatusNotFound, fmt.Errorf("machine %s not found", machineID))
			return
		}
		respondError(w, http.StatusInternalServerError, err)
		return
	}

	rendered, err := a.renderer.Render("unattend.xml.tmpl", map[string]any{
		"Machine": machine,
		"Profile": machine.Profile,
	})
	if err != nil {
		respondError(w, http.StatusInternalServerError, err)
		return
	}

	w.Header().Set("Content-Type", "application/xml")
	_, _ = w.Write([]byte(rendered))
}

func (a *API) handleArtifacts(w http.ResponseWriter, r *http.Request) {
	if a.store.S3 == nil {
		respondError(w, http.StatusFailedDependency, errors.New("s3 client not configured"))
		return
	}

	var req struct {
		Kind   string         `json:"kind"`
		SHA256 string         `json:"sha256"`
		Meta   map[string]any `json:"meta"`
	}
	if err := decodeJSON(r, &req); err != nil {
		respondError(w, http.StatusBadRequest, err)
		return
	}

	req.Kind = strings.TrimSpace(req.Kind)
	req.SHA256 = strings.TrimSpace(req.SHA256)
	if req.Kind == "" || req.SHA256 == "" {
		respondError(w, http.StatusBadRequest, errors.New("kind and sha256 are required"))
		return
	}
	if req.Meta == nil {
		req.Meta = map[string]any{}
	}

	artifactID := uuid.New()
	key := fmt.Sprintf("artifacts/%s/%s", req.Kind, artifactID)
	location := fmt.Sprintf("s3://%s/%s", a.config.ArtifactBucket, key)
	createdAt := time.Now().UTC()

	metaBytes, err := json.Marshal(req.Meta)
	if err != nil {
		respondError(w, http.StatusBadRequest, fmt.Errorf("marshal meta: %w", err))
		return
	}

	query := `
        INSERT INTO artifacts (id, kind, sha256, url, meta, created_at)
        VALUES ($1, $2, $3, $4, $5::jsonb, $6)
        RETURNING id, kind, sha256, url, meta, created_at;
    `

	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	row := a.store.DB.QueryRow(ctx, query, artifactID, req.Kind, req.SHA256, location, string(metaBytes), createdAt)
	var art Artifact
	var metaRaw []byte
	if err := row.Scan(&art.ID, &art.Kind, &art.SHA256, &art.URL, &metaRaw, &art.CreatedAt); err != nil {
		respondError(w, http.StatusInternalServerError, err)
		return
	}
	if len(metaRaw) > 0 {
		if err := json.Unmarshal(metaRaw, &art.Meta); err != nil {
			respondError(w, http.StatusInternalServerError, fmt.Errorf("decode meta: %w", err))
			return
		}
	} else {
		art.Meta = map[string]any{}
	}

	uploadURL, err := a.store.S3.PresignPut(ctx, a.config.ArtifactBucket, key, presignURLExpiry)
	if err != nil {
		respondError(w, http.StatusInternalServerError, fmt.Errorf("presign put: %w", err))
		return
	}

	respondJSON(w, http.StatusCreated, map[string]any{
		"artifact":   art,
		"upload_url": uploadURL,
	})
}

func (a *API) handleFacts(w http.ResponseWriter, r *http.Request) {
	var req struct {
		MachineID uuid.UUID      `json:"machine_id"`
		Snapshot  map[string]any `json:"snapshot"`
	}
	if err := decodeJSON(r, &req); err != nil {
		respondError(w, http.StatusBadRequest, err)
		return
	}
	if req.MachineID == uuid.Nil {
		respondError(w, http.StatusBadRequest, errors.New("machine_id is required"))
		return
	}
	if req.Snapshot == nil {
		req.Snapshot = map[string]any{}
	}

	snapshotBytes, err := json.Marshal(req.Snapshot)
	if err != nil {
		respondError(w, http.StatusBadRequest, fmt.Errorf("marshal snapshot: %w", err))
		return
	}

	factID := uuid.New()
	createdAt := time.Now().UTC()

	query := `
        INSERT INTO facts (id, machine_id, snapshot, created_at)
        VALUES ($1, $2, $3::jsonb, $4)
        RETURNING id, machine_id, snapshot, created_at;
    `

	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	row := a.store.DB.QueryRow(ctx, query, factID, req.MachineID, string(snapshotBytes), createdAt)
	var fact struct {
		ID        uuid.UUID `json:"id"`
		MachineID uuid.UUID `json:"machine_id"`
		Snapshot  []byte    `json:"-"`
		CreatedAt time.Time `json:"created_at"`
	}
	if err := row.Scan(&fact.ID, &fact.MachineID, &fact.Snapshot, &fact.CreatedAt); err != nil {
		respondError(w, http.StatusInternalServerError, err)
		return
	}

	var snapshot map[string]any
	if len(fact.Snapshot) > 0 {
		if err := json.Unmarshal(fact.Snapshot, &snapshot); err != nil {
			respondError(w, http.StatusInternalServerError, fmt.Errorf("decode snapshot: %w", err))
			return
		}
	} else {
		snapshot = map[string]any{}
	}

	a.publishJSON(agentFactsTopic, map[string]any{
		"fact_id":    fact.ID,
		"machine_id": fact.MachineID,
		"snapshot":   snapshot,
		"created_at": fact.CreatedAt,
	})

	respondJSON(w, http.StatusCreated, map[string]any{
		"fact": map[string]any{
			"id":         fact.ID,
			"machine_id": fact.MachineID,
			"snapshot":   snapshot,
			"created_at": fact.CreatedAt,
		},
	})
}

func (a *API) handleRunStart(w http.ResponseWriter, r *http.Request) {
	var req struct {
		MachineID   uuid.UUID `json:"machine_id"`
		BlueprintID uuid.UUID `json:"blueprint_id"`
	}
	if err := decodeJSON(r, &req); err != nil {
		respondError(w, http.StatusBadRequest, err)
		return
	}
	if req.MachineID == uuid.Nil {
		respondError(w, http.StatusBadRequest, errors.New("machine_id is required"))
		return
	}
	if req.BlueprintID == uuid.Nil {
		respondError(w, http.StatusBadRequest, errors.New("blueprint_id is required"))
		return
	}

	runID := uuid.New()
	now := time.Now().UTC()
	status := "running"

	query := `
        INSERT INTO runs (id, machine_id, blueprint_id, status, started_at)
        VALUES ($1, $2, $3, $4, $5)
        RETURNING id, machine_id, blueprint_id, status, started_at, finished_at, logs;
    `

	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	row := a.store.DB.QueryRow(ctx, query, runID, req.MachineID, req.BlueprintID, status, now)
	var run Run
	if err := row.Scan(&run.ID, &run.MachineID, &run.BlueprintID, &run.Status, &run.StartedAt, &run.FinishedAt, &run.Logs); err != nil {
		respondError(w, http.StatusInternalServerError, err)
		return
	}

	a.publishJSON(runStartedTopic, map[string]any{
		"run_id":       run.ID,
		"machine_id":   run.MachineID,
		"blueprint_id": run.BlueprintID,
		"status":       run.Status,
		"started_at":   run.StartedAt,
	})

	respondJSON(w, http.StatusCreated, map[string]any{"run": run})
}

func (a *API) handleRunFinish(w http.ResponseWriter, r *http.Request) {
	var req struct {
		RunID  uuid.UUID `json:"run_id"`
		Status string    `json:"status"`
		Logs   string    `json:"logs"`
	}
	if err := decodeJSON(r, &req); err != nil {
		respondError(w, http.StatusBadRequest, err)
		return
	}
	if req.RunID == uuid.Nil {
		respondError(w, http.StatusBadRequest, errors.New("run_id is required"))
		return
	}
	req.Status = strings.TrimSpace(req.Status)
	if req.Status == "" {
		respondError(w, http.StatusBadRequest, errors.New("status is required"))
		return
	}

	now := time.Now().UTC()

	query := `
        UPDATE runs
        SET status = $2, logs = $3, finished_at = $4
        WHERE id = $1
        RETURNING id, machine_id, blueprint_id, status, started_at, finished_at, logs;
    `

	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	row := a.store.DB.QueryRow(ctx, query, req.RunID, req.Status, req.Logs, now)
	var run Run
	if err := row.Scan(&run.ID, &run.MachineID, &run.BlueprintID, &run.Status, &run.StartedAt, &run.FinishedAt, &run.Logs); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			respondError(w, http.StatusNotFound, fmt.Errorf("run %s not found", req.RunID))
			return
		}
		respondError(w, http.StatusInternalServerError, err)
		return
	}

	a.publishJSON(runFinishedTopic, map[string]any{
		"run_id":       run.ID,
		"machine_id":   run.MachineID,
		"blueprint_id": run.BlueprintID,
		"status":       run.Status,
		"finished_at":  run.FinishedAt,
	})

	respondJSON(w, http.StatusOK, map[string]any{"run": run})
}

func (a *API) fetchMachineByMAC(ctx context.Context, mac string) (Machine, error) {
	query := `
        SELECT id, mac, serial, profile, created_at, updated_at
        FROM machines
        WHERE mac = $1
    `

	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	var m Machine
	var profileRaw []byte
	err := a.store.DB.QueryRow(ctx, query, mac).Scan(&m.ID, &m.MAC, &m.Serial, &profileRaw, &m.CreatedAt, &m.UpdatedAt)
	if err != nil {
		return Machine{}, err
	}
	if len(profileRaw) > 0 {
		if err := json.Unmarshal(profileRaw, &m.Profile); err != nil {
			return Machine{}, fmt.Errorf("decode profile: %w", err)
		}
	} else {
		m.Profile = map[string]any{}
	}
	return m, nil
}

func (a *API) fetchMachineByID(ctx context.Context, id uuid.UUID) (Machine, error) {
	query := `
        SELECT id, mac, serial, profile, created_at, updated_at
        FROM machines
        WHERE id = $1
    `

	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	var m Machine
	var profileRaw []byte
	err := a.store.DB.QueryRow(ctx, query, id).Scan(&m.ID, &m.MAC, &m.Serial, &profileRaw, &m.CreatedAt, &m.UpdatedAt)
	if err != nil {
		return Machine{}, err
	}
	if len(profileRaw) > 0 {
		if err := json.Unmarshal(profileRaw, &m.Profile); err != nil {
			return Machine{}, fmt.Errorf("decode profile: %w", err)
		}
	} else {
		m.Profile = map[string]any{}
	}
	return m, nil
}

func (a *API) publishJSON(subject string, payload map[string]any) {
	if a.store.Bus == nil || subject == "" {
		return
	}
	data, err := json.Marshal(payload)
	if err != nil {
		return
	}
	_ = a.store.Bus.Publish(subject, data)
}

func decodeJSON(r *http.Request, dest any) error {
	if r.Body == nil {
		return errors.New("request body required")
	}
	defer r.Body.Close()
	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()
	return dec.Decode(dest)
}

func respondJSON(w http.ResponseWriter, status int, payload any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if payload == nil {
		return
	}
	_ = json.NewEncoder(w).Encode(payload)
}

func respondError(w http.ResponseWriter, status int, err error) {
	if err == nil {
		err = errors.New("unknown error")
	}
	respondJSON(w, status, map[string]any{"error": err.Error()})
}
