package api

import (
	"time"

	"github.com/google/uuid"
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
