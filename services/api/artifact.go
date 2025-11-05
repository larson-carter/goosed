package api

import (
	"time"

	"github.com/google/uuid"
)

// Artifact tracks files uploaded by agents or the control plane for later consumption.
type Artifact struct {
	ID        uuid.UUID      `json:"id" db:"id"`
	Kind      string         `json:"kind" db:"kind"`
	SHA256    string         `json:"sha256" db:"sha256"`
	URL       string         `json:"url" db:"url"`
	Meta      map[string]any `json:"meta" db:"meta"`
	CreatedAt time.Time      `json:"created_at" db:"created_at"`
}
