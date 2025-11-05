package api

import (
	"errors"
	"net/http"
	"os"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"

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
	if store.ORM == nil {
		return nil, errors.New("store ORM is required")
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

	tokenStore, err := newTokenStore(store.ORM, cfg.TokenTTL)
	if err != nil {
		return nil, err
	}

	return &API{
		store:    store,
		renderer: renderer,
		config:   cfg,
		tokens:   tokenStore,
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
		r.Post("/agents/token/refresh", a.handleAgentTokenRefresh)
		r.Post("/runs/start", a.handleRunStart)
		r.Post("/runs/finish", a.handleRunFinish)
	})

	return r, nil
}
