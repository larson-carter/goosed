package api

import (
	"errors"
	"net/http"
	"time"

	"github.com/google/uuid"
)

func (a *API) handleFacts(w http.ResponseWriter, r *http.Request) {
	var req struct {
		MachineID uuid.UUID      `json:"machine_id"`
		Snapshot  map[string]any `json:"snapshot"`
	}
	if err := decodeJSON(r, &req); err != nil {
		respondError(w, http.StatusBadRequest, err)
		return
	}
	if req.MachineID == uuid.Nil {
		respondError(w, http.StatusBadRequest, errors.New("machine_id is required"))
		return
	}
	if req.Snapshot == nil {
		req.Snapshot = map[string]any{}
	}

	ctx, cancel := withTimeout(r.Context())
	defer cancel()

	model := factModel{
		ID:        uuid.New(),
		MachineID: req.MachineID,
		Snapshot:  toJSONMap(req.Snapshot),
		CreatedAt: time.Now().UTC(),
	}

	if err := a.store.ORM.WithContext(ctx).Create(&model).Error; err != nil {
		respondError(w, http.StatusInternalServerError, err)
		return
	}

	fact := model.toMap()

	a.publishJSON(agentFactsTopic, map[string]any{
		"fact_id":    fact["id"],
		"machine_id": fact["machine_id"],
		"snapshot":   fact["snapshot"],
		"created_at": fact["created_at"],
	})

	respondJSON(w, http.StatusCreated, map[string]any{"fact": fact})
}
