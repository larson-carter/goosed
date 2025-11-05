package api

import (
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/google/uuid"
)

func (a *API) handleArtifacts(w http.ResponseWriter, r *http.Request) {
	if a.store.S3 == nil {
		respondError(w, http.StatusFailedDependency, errors.New("s3 client not configured"))
		return
	}

	var req struct {
		Kind   string         `json:"kind"`
		SHA256 string         `json:"sha256"`
		Meta   map[string]any `json:"meta"`
	}
	if err := decodeJSON(r, &req); err != nil {
		respondError(w, http.StatusBadRequest, err)
		return
	}

	req.Kind = strings.TrimSpace(req.Kind)
	req.SHA256 = strings.TrimSpace(req.SHA256)
	if req.Kind == "" || req.SHA256 == "" {
		respondError(w, http.StatusBadRequest, errors.New("kind and sha256 are required"))
		return
	}
	if req.Meta == nil {
		req.Meta = map[string]any{}
	}

	ctx, cancel := withTimeout(r.Context())
	defer cancel()

	artifactID := uuid.New()
	key := fmt.Sprintf("artifacts/%s/%s", req.Kind, artifactID)
	location := fmt.Sprintf("s3://%s/%s", a.config.ArtifactBucket, key)
	now := time.Now().UTC()

	model := artifactModel{
		ID:        artifactID,
		Kind:      req.Kind,
		SHA256:    req.SHA256,
		URL:       location,
		Meta:      toJSONMap(req.Meta),
		CreatedAt: now,
	}

	orm := a.store.ORM.WithContext(ctx)
	if err := orm.Create(&model).Error; err != nil {
		respondError(w, http.StatusInternalServerError, err)
		return
	}

	artifact := model.toAPI()

	uploadURL, err := a.store.S3.PresignPut(ctx, a.config.ArtifactBucket, key, presignURLExpiry)
	if err != nil {
		respondError(w, http.StatusInternalServerError, fmt.Errorf("presign put: %w", err))
		return
	}

	respondJSON(w, http.StatusCreated, map[string]any{
		"artifact":   artifact,
		"upload_url": uploadURL,
	})
}
