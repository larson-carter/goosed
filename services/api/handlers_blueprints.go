package api

import (
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

func (a *API) handleListBlueprints(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := withTimeout(r.Context())
	defer cancel()

	orm := a.store.ORM.WithContext(ctx)

	var models []blueprintModel
	if err := orm.Order("name ASC").Find(&models).Error; err != nil {
		respondError(w, http.StatusInternalServerError, err)
		return
	}

	items := make([]Blueprint, 0, len(models))
	for _, model := range models {
		items = append(items, model.toAPI())
	}

	respondJSON(w, http.StatusOK, map[string]any{"blueprints": items})
}

func (a *API) handleCreateBlueprint(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Name    string         `json:"name"`
		OS      string         `json:"os"`
		Version string         `json:"version"`
		Data    map[string]any `json:"data"`
	}
	if err := decodeJSON(r, &req); err != nil {
		respondError(w, http.StatusBadRequest, err)
		return
	}

	req.Name = strings.TrimSpace(req.Name)
	req.OS = strings.TrimSpace(req.OS)
	req.Version = strings.TrimSpace(req.Version)
	if req.Name == "" {
		respondError(w, http.StatusBadRequest, errors.New("name is required"))
		return
	}
	if req.OS == "" {
		respondError(w, http.StatusBadRequest, errors.New("os is required"))
		return
	}
	if req.Version == "" {
		respondError(w, http.StatusBadRequest, errors.New("version is required"))
		return
	}
	if req.Data == nil {
		req.Data = map[string]any{}
	}

	ctx, cancel := withTimeout(r.Context())
	defer cancel()

	now := time.Now().UTC()
	model := blueprintModel{
		ID:        uuid.New(),
		Name:      req.Name,
		OS:        req.OS,
		Version:   req.Version,
		Data:      toJSONMap(req.Data),
		CreatedAt: now,
		UpdatedAt: now,
	}

	if err := a.store.ORM.WithContext(ctx).Create(&model).Error; err != nil {
		respondError(w, http.StatusInternalServerError, err)
		return
	}

	respondJSON(w, http.StatusCreated, map[string]any{"blueprint": model.toAPI()})
}

func (a *API) handleGetBlueprint(w http.ResponseWriter, r *http.Request) {
	idParam := strings.TrimSpace(chi.URLParam(r, "blueprintID"))
	id, err := uuid.Parse(idParam)
	if err != nil {
		respondError(w, http.StatusBadRequest, errors.New("invalid blueprint id"))
		return
	}

	ctx, cancel := withTimeout(r.Context())
	defer cancel()

	var model blueprintModel
	switch err := a.store.ORM.WithContext(ctx).First(&model, "id = ?", id).Error; {
	case errors.Is(err, gorm.ErrRecordNotFound):
		respondError(w, http.StatusNotFound, errors.New("blueprint not found"))
		return
	case err != nil:
		respondError(w, http.StatusInternalServerError, err)
		return
	}

	respondJSON(w, http.StatusOK, map[string]any{"blueprint": model.toAPI()})
}

func (a *API) handleUpdateBlueprint(w http.ResponseWriter, r *http.Request) {
	idParam := strings.TrimSpace(chi.URLParam(r, "blueprintID"))
	id, err := uuid.Parse(idParam)
	if err != nil {
		respondError(w, http.StatusBadRequest, errors.New("invalid blueprint id"))
		return
	}

	var req struct {
		Name    string         `json:"name"`
		OS      string         `json:"os"`
		Version string         `json:"version"`
		Data    map[string]any `json:"data"`
	}
	if err := decodeJSON(r, &req); err != nil {
		respondError(w, http.StatusBadRequest, err)
		return
	}

	req.Name = strings.TrimSpace(req.Name)
	req.OS = strings.TrimSpace(req.OS)
	req.Version = strings.TrimSpace(req.Version)
	if req.Name == "" {
		respondError(w, http.StatusBadRequest, errors.New("name is required"))
		return
	}
	if req.OS == "" {
		respondError(w, http.StatusBadRequest, errors.New("os is required"))
		return
	}
	if req.Version == "" {
		respondError(w, http.StatusBadRequest, errors.New("version is required"))
		return
	}
	if req.Data == nil {
		req.Data = map[string]any{}
	}

	ctx, cancel := withTimeout(r.Context())
	defer cancel()

	orm := a.store.ORM.WithContext(ctx)

	var existing blueprintModel
	switch err := orm.First(&existing, "id = ?", id).Error; {
	case errors.Is(err, gorm.ErrRecordNotFound):
		respondError(w, http.StatusNotFound, errors.New("blueprint not found"))
		return
	case err != nil:
		respondError(w, http.StatusInternalServerError, err)
		return
	}

	updates := map[string]any{
		"name":       req.Name,
		"os":         req.OS,
		"version":    req.Version,
		"data":       toJSONMap(req.Data),
		"updated_at": time.Now().UTC(),
	}

	if err := orm.Model(&existing).Updates(updates).Error; err != nil {
		respondError(w, http.StatusInternalServerError, err)
		return
	}

	if err := orm.First(&existing, "id = ?", id).Error; err != nil {
		respondError(w, http.StatusInternalServerError, err)
		return
	}

	respondJSON(w, http.StatusOK, map[string]any{"blueprint": existing.toAPI()})
}

func (a *API) handleDeleteBlueprint(w http.ResponseWriter, r *http.Request) {
	idParam := strings.TrimSpace(chi.URLParam(r, "blueprintID"))
	id, err := uuid.Parse(idParam)
	if err != nil {
		respondError(w, http.StatusBadRequest, errors.New("invalid blueprint id"))
		return
	}

	ctx, cancel := withTimeout(r.Context())
	defer cancel()

	orm := a.store.ORM.WithContext(ctx)

	result := orm.Delete(&blueprintModel{}, "id = ?", id)
	if result.Error != nil {
		respondError(w, http.StatusInternalServerError, result.Error)
		return
	}
	if result.RowsAffected == 0 {
		respondError(w, http.StatusNotFound, errors.New("blueprint not found"))
		return
	}

	respondJSON(w, http.StatusNoContent, nil)
}
