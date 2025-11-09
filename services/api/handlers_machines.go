package api

import (
	"context"
	"errors"
	"net/http"
	"sort"
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

func (a *API) handleListMachines(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := withTimeout(r.Context())
	defer cancel()

	orm := a.store.ORM.WithContext(ctx)

	var models []machineModel
	if err := orm.Order("created_at DESC").Find(&models).Error; err != nil {
		respondError(w, http.StatusInternalServerError, err)
		return
	}

	if len(models) == 0 {
		respondJSON(w, http.StatusOK, map[string]any{"machines": []machineListItem{}})
		return
	}

	machineIDs := make([]uuid.UUID, 0, len(models))
	for _, model := range models {
		machineIDs = append(machineIDs, model.ID)
	}

	latestFacts, err := a.fetchLatestFacts(ctx, machineIDs)
	if err != nil {
		respondError(w, http.StatusInternalServerError, err)
		return
	}

	recentRuns, err := a.fetchRecentRuns(ctx, machineIDs, 5)
	if err != nil {
		respondError(w, http.StatusInternalServerError, err)
		return
	}

	items := make([]machineListItem, 0, len(models))
	for _, model := range models {
		machine := model.toAPI()

		fact, _ := latestFacts[machine.ID]
		runs := recentRuns[machine.ID]

		status := deriveMachineStatus(fact, runs)

		items = append(items, machineListItem{
			Machine:    machine,
			Status:     status,
			LatestFact: fact,
			RecentRuns: runs,
		})
	}

	respondJSON(w, http.StatusOK, map[string]any{"machines": items})
}

func (a *API) fetchLatestFacts(ctx context.Context, machineIDs []uuid.UUID) (map[uuid.UUID]*machineListFact, error) {
	if len(machineIDs) == 0 {
		return map[uuid.UUID]*machineListFact{}, nil
	}

	ctx, cancel := withTimeout(ctx)
	defer cancel()

	var facts []factModel
	if err := a.store.ORM.WithContext(ctx).
		Where("machine_id IN ?", machineIDs).
		Order("created_at DESC").
		Find(&facts).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return map[uuid.UUID]*machineListFact{}, nil
		}
		return nil, err
	}

	out := make(map[uuid.UUID]*machineListFact, len(machineIDs))
	for _, fact := range facts {
		if _, ok := out[fact.MachineID]; ok {
			continue
		}
		snapshot := mapFromJSONMap(fact.Snapshot)
		out[fact.MachineID] = &machineListFact{
			ID:        fact.ID,
			Snapshot:  snapshot,
			CreatedAt: fact.CreatedAt,
		}
	}
	return out, nil
}

func (a *API) fetchRecentRuns(ctx context.Context, machineIDs []uuid.UUID, perMachine int) (map[uuid.UUID][]Run, error) {
	if len(machineIDs) == 0 || perMachine <= 0 {
		return map[uuid.UUID][]Run{}, nil
	}

	ctx, cancel := withTimeout(ctx)
	defer cancel()

	var models []runModel
	if err := a.store.ORM.WithContext(ctx).
		Where("machine_id IN ?", machineIDs).
		Order("started_at DESC NULLS LAST, created_at DESC").
		Find(&models).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return map[uuid.UUID][]Run{}, nil
		}
		return nil, err
	}

	runsByMachine := make(map[uuid.UUID][]Run, len(machineIDs))
	for _, model := range models {
		if model.MachineID == nil || *model.MachineID == uuid.Nil {
			continue
		}
		id := *model.MachineID
		if len(runsByMachine[id]) >= perMachine {
			continue
		}
		run := model.toAPI()
		runsByMachine[id] = append(runsByMachine[id], run)
	}

	for id := range runsByMachine {
		runs := runsByMachine[id]
		sort.SliceStable(runs, func(i, j int) bool {
			ti := runTime(runs[i])
			tj := runTime(runs[j])
			return tj.Before(ti)
		})
		if len(runs) > perMachine {
			runsByMachine[id] = runs[:perMachine]
		}
	}

	return runsByMachine, nil
}

func runTime(run Run) time.Time {
	if run.StartedAt != nil {
		return run.StartedAt.In(time.UTC)
	}
	if run.FinishedAt != nil {
		return run.FinishedAt.In(time.UTC)
	}
	return time.Time{}
}

func deriveMachineStatus(fact *machineListFact, runs []Run) string {
	if len(runs) > 0 {
		latest := runs[0]
		switch strings.ToLower(strings.TrimSpace(latest.Status)) {
		case "running":
			return machineStatusProvisioning
		case "success", "succeeded", "completed":
			return machineStatusReady
		case "failed", "failure", "error", "errored":
			return machineStatusError
		}
	}

	if fact != nil {
		return machineStatusReady
	}

	return machineStatusOffline
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
