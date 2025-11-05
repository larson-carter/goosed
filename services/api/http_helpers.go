package api

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"time"
)

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

func withTimeout(ctx context.Context) (context.Context, context.CancelFunc) {
	return context.WithTimeout(ctx, 5*time.Second)
}
