package api

import (
	"context"
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

func (a *API) handleUpsertMachine(w http.ResponseWriter, r *http.Request) {
	var req struct {
		MAC     string         `json:"mac"`
		Serial  string         `json:"serial"`
		Profile map[string]any `json:"profile"`
	}
	if err := decodeJSON(r, &req); err != nil {
		respondError(w, http.StatusBadRequest, err)
		return
	}

	req.MAC = strings.ToLower(strings.TrimSpace(req.MAC))
	if req.MAC == "" {
		respondError(w, http.StatusBadRequest, errors.New("mac is required"))
		return
	}
	if req.Profile == nil {
		req.Profile = map[string]any{}
	}

	ctx, cancel := withTimeout(r.Context())
	defer cancel()

	orm := a.store.ORM.WithContext(ctx)

	var existing machineModel
	err := orm.Where("mac = ?", req.MAC).First(&existing).Error
	switch {
	case errors.Is(err, gorm.ErrRecordNotFound):
		now := time.Now().UTC()
		model := machineModel{
			ID:        uuid.New(),
			MAC:       req.MAC,
			Serial:    req.Serial,
			Profile:   toJSONMap(req.Profile),
			CreatedAt: now,
			UpdatedAt: now,
		}
		if err := orm.Create(&model).Error; err != nil {
			respondError(w, http.StatusInternalServerError, err)
			return
		}
		machine := model.toAPI()
		a.publishJSON(machinesTopic, map[string]any{
			"machine_id": machine.ID,
			"mac":        machine.MAC,
		})
		respondJSON(w, http.StatusOK, map[string]any{"machine": machine})
	case err != nil:
		respondError(w, http.StatusInternalServerError, err)
	default:
		updates := map[string]any{
			"serial":     req.Serial,
			"profile":    toJSONMap(req.Profile),
			"updated_at": time.Now().UTC(),
		}
		if err := orm.Model(&existing).Updates(updates).Error; err != nil {
			respondError(w, http.StatusInternalServerError, err)
			return
		}

		if err := orm.First(&existing, "id = ?", existing.ID).Error; err != nil {
			respondError(w, http.StatusInternalServerError, err)
			return
		}

		machine := existing.toAPI()
		a.publishJSON(machinesTopic, map[string]any{
			"machine_id": machine.ID,
			"mac":        machine.MAC,
		})
		respondJSON(w, http.StatusOK, map[string]any{"machine": machine})
	}
}

func (a *API) fetchMachineByMAC(ctx context.Context, mac string) (Machine, error) {
	ctx, cancel := withTimeout(ctx)
	defer cancel()

	var model machineModel
	if err := a.store.ORM.WithContext(ctx).Where("mac = ?", mac).First(&model).Error; err != nil {
		return Machine{}, err
	}
	return model.toAPI(), nil
}

func (a *API) fetchMachineByID(ctx context.Context, id uuid.UUID) (Machine, error) {
	ctx, cancel := withTimeout(ctx)
	defer cancel()

	var model machineModel
	if err := a.store.ORM.WithContext(ctx).Where("id = ?", id).First(&model).Error; err != nil {
		return Machine{}, err
	}
	return model.toAPI(), nil
}
