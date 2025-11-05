package api

import (
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type tokenRefreshRequest struct {
	MachineID string `json:"machine_id"`
	OldToken  string `json:"old_token"`
}

func (a *API) handleAgentTokenRefresh(w http.ResponseWriter, r *http.Request) {
	var req tokenRefreshRequest
	if err := decodeJSON(r, &req); err != nil {
		respondError(w, http.StatusBadRequest, err)
		return
	}

	machineID, err := uuid.Parse(strings.TrimSpace(req.MachineID))
	if err != nil {
		respondError(w, http.StatusBadRequest, errors.New("valid machine_id is required"))
		return
	}

	oldToken := strings.TrimSpace(req.OldToken)
	if oldToken == "" {
		respondError(w, http.StatusBadRequest, errors.New("old_token is required"))
		return
	}

	machine, err := a.fetchMachineByID(r.Context(), machineID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			respondError(w, http.StatusNotFound, errors.New("machine not found"))
			return
		}
		respondError(w, http.StatusInternalServerError, err)
		return
	}

	rotated, err := a.tokens.Rotate(r.Context(), machine.MAC, oldToken)
	if err != nil {
		switch {
		case errors.Is(err, ErrTokenNotFound):
			respondError(w, http.StatusUnauthorized, errors.New("invalid token"))
			return
		case errors.Is(err, ErrTokenExpired):
			respondError(w, http.StatusUnauthorized, errors.New("token expired"))
			return
		default:
			respondError(w, http.StatusInternalServerError, err)
			return
		}
	}

	respondJSON(w, http.StatusOK, map[string]any{
		"token":      rotated.Value,
		"expires_at": rotated.ExpiresAt.Format(time.RFC3339),
	})
}
