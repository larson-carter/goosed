package api

import (
	"sync"
	"time"

	"github.com/google/uuid"
)

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

	now := time.Now()
	for key, entry := range ts.tokens {
		if now.After(entry.Expires) {
			delete(ts.tokens, key)
		}
	}

	token := uuid.New().String()
	ts.tokens[token] = tokenEntry{MAC: mac, Expires: now.Add(ts.ttl)}
	return token
}
