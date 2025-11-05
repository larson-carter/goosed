package api

import (
	"time"

	"github.com/google/uuid"
)

// Token represents a boot or agent authentication token tracked by the store.
type Token struct {
	ID        uuid.UUID
	MAC       string
	Value     string
	ExpiresAt time.Time
	Used      bool
}
