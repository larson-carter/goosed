package api

import (
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

func (a *API) handleRunStart(w http.ResponseWriter, r *http.Request) {
	var req struct {
		MachineID   uuid.UUID `json:"machine_id"`
		BlueprintID uuid.UUID `json:"blueprint_id"`
	}
	if err := decodeJSON(r, &req); err != nil {
		respondError(w, http.StatusBadRequest, err)
		return
	}
	if req.MachineID == uuid.Nil {
		respondError(w, http.StatusBadRequest, errors.New("machine_id is required"))
		return
	}
	if req.BlueprintID == uuid.Nil {
		respondError(w, http.StatusBadRequest, errors.New("blueprint_id is required"))
		return
	}

	ctx, cancel := withTimeout(r.Context())
	defer cancel()

	now := time.Now().UTC()
	model := runModel{
		ID:          uuid.New(),
		MachineID:   &req.MachineID,
		BlueprintID: &req.BlueprintID,
		Status:      "running",
		StartedAt:   &now,
	}

	if err := a.store.ORM.WithContext(ctx).Create(&model).Error; err != nil {
		respondError(w, http.StatusInternalServerError, err)
		return
	}

	run := model.toAPI()

	a.publishJSON(runStartedTopic, map[string]any{
		"run_id":       run.ID,
		"machine_id":   run.MachineID,
		"blueprint_id": run.BlueprintID,
		"status":       run.Status,
		"started_at":   run.StartedAt,
	})

	respondJSON(w, http.StatusCreated, map[string]any{"run": run})
}

func (a *API) handleRunFinish(w http.ResponseWriter, r *http.Request) {
	var req struct {
		RunID  uuid.UUID `json:"run_id"`
		Status string    `json:"status"`
		Logs   string    `json:"logs"`
	}
	if err := decodeJSON(r, &req); err != nil {
		respondError(w, http.StatusBadRequest, err)
		return
	}
	if req.RunID == uuid.Nil {
		respondError(w, http.StatusBadRequest, errors.New("run_id is required"))
		return
	}
	req.Status = strings.TrimSpace(req.Status)
	if req.Status == "" {
		respondError(w, http.StatusBadRequest, errors.New("status is required"))
		return
	}

	ctx, cancel := withTimeout(r.Context())
	defer cancel()

	orm := a.store.ORM.WithContext(ctx)

	var model runModel
	if err := orm.First(&model, "id = ?", req.RunID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			respondError(w, http.StatusNotFound, fmt.Errorf("run %s not found", req.RunID))
			return
		}
		respondError(w, http.StatusInternalServerError, err)
		return
	}

	finishedAt := time.Now().UTC()
	model.Status = req.Status
	model.Logs = req.Logs
	model.FinishedAt = &finishedAt

	if err := orm.Save(&model).Error; err != nil {
		respondError(w, http.StatusInternalServerError, err)
		return
	}

	run := model.toAPI()

	a.publishJSON(runFinishedTopic, map[string]any{
		"run_id":       run.ID,
		"machine_id":   run.MachineID,
		"blueprint_id": run.BlueprintID,
		"status":       run.Status,
		"finished_at":  run.FinishedAt,
	})

	respondJSON(w, http.StatusOK, map[string]any{"run": run})
}
