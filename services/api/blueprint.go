package api

import "github.com/google/uuid"

// Blueprint describes desired OS/application configuration payloads.
type Blueprint struct {
	ID      uuid.UUID      `json:"id" db:"id"`
	Name    string         `json:"name" db:"name"`
	OS      string         `json:"os" db:"os"`
	Version string         `json:"version" db:"version"`
	Data    map[string]any `json:"data" db:"data"`
}
