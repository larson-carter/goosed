package api

import (
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/nats-io/nats.go"
	"gorm.io/gorm"

	gos3 "goosed/pkg/s3"
)

// Machine models a provisionable host in the system.
type Machine struct {
	ID        uuid.UUID      `json:"id" db:"id"`
	MAC       string         `json:"mac" db:"mac"`
	Serial    string         `json:"serial" db:"serial"`
	Profile   map[string]any `json:"profile" db:"profile"`
	CreatedAt time.Time      `json:"created_at" db:"created_at"`
	UpdatedAt time.Time      `json:"updated_at" db:"updated_at"`
}

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

// Artifact tracks files uploaded by agents or the control plane for later consumption.
type Artifact struct {
	ID        uuid.UUID      `json:"id" db:"id"`
	Kind      string         `json:"kind" db:"kind"`
	SHA256    string         `json:"sha256" db:"sha256"`
	URL       string         `json:"url" db:"url"`
	Meta      map[string]any `json:"meta" db:"meta"`
	CreatedAt time.Time      `json:"created_at" db:"created_at"`
}

// Blueprint describes desired OS/application configuration payloads.
type Blueprint struct {
	ID      uuid.UUID      `json:"id" db:"id"`
	Name    string         `json:"name" db:"name"`
	OS      string         `json:"os" db:"os"`
	Version string         `json:"version" db:"version"`
	Data    map[string]any `json:"data" db:"data"`
}

// Store holds external dependencies required by the API layer.
type Store struct {
	DB  *pgxpool.Pool
	ORM *gorm.DB
	S3  *gos3.Client
	Bus *nats.Conn
}
