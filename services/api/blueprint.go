package api

import (
	"time"

	"github.com/google/uuid"
)

// Blueprint describes desired OS/application configuration payloads.
type Blueprint struct {
	ID        uuid.UUID      `json:"id" db:"id"`
	Name      string         `json:"name" db:"name"`
	OS        string         `json:"os" db:"os"`
	Version   string         `json:"version" db:"version"`
	Data      map[string]any `json:"data" db:"data"`
	CreatedAt time.Time      `json:"created_at" db:"created_at"`
	UpdatedAt time.Time      `json:"updated_at" db:"updated_at"`
}
