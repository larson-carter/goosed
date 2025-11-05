package api

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/datatypes"
)

type machineModel struct {
	ID        uuid.UUID         `gorm:"type:uuid;primaryKey"`
	MAC       string            `gorm:"type:text;uniqueIndex;not null"`
	Serial    string            `gorm:"type:text"`
	Profile   datatypes.JSONMap `gorm:"type:jsonb"`
	CreatedAt time.Time         `gorm:"type:timestamptz;not null;default:now();autoCreateTime"`
	UpdatedAt time.Time         `gorm:"type:timestamptz;not null;default:now();autoUpdateTime"`
}

func (machineModel) TableName() string { return "machines" }

func (m machineModel) toAPI() Machine {
	return Machine{
		ID:        m.ID,
		MAC:       m.MAC,
		Serial:    m.Serial,
		Profile:   mapFromJSONMap(m.Profile),
		CreatedAt: m.CreatedAt,
		UpdatedAt: m.UpdatedAt,
	}
}

type runModel struct {
	ID          uuid.UUID  `gorm:"type:uuid;primaryKey"`
	MachineID   *uuid.UUID `gorm:"type:uuid"`
	BlueprintID *uuid.UUID `gorm:"type:uuid"`
	Status      string     `gorm:"type:text"`
	StartedAt   *time.Time `gorm:"type:timestamptz"`
	FinishedAt  *time.Time `gorm:"type:timestamptz"`
	Logs        string     `gorm:"type:text"`
}

func (runModel) TableName() string { return "runs" }

func (r runModel) toAPI() Run {
	return Run{
		ID:          r.ID,
		MachineID:   valueOrZero(r.MachineID),
		BlueprintID: valueOrZero(r.BlueprintID),
		Status:      r.Status,
		StartedAt:   r.StartedAt,
		FinishedAt:  r.FinishedAt,
		Logs:        r.Logs,
	}
}

type artifactModel struct {
	ID        uuid.UUID         `gorm:"type:uuid;primaryKey"`
	Kind      string            `gorm:"type:text;not null"`
	SHA256    string            `gorm:"type:text;not null"`
	URL       string            `gorm:"type:text;not null"`
	Meta      datatypes.JSONMap `gorm:"type:jsonb"`
	CreatedAt time.Time         `gorm:"type:timestamptz;not null;default:now();autoCreateTime"`
}

func (artifactModel) TableName() string { return "artifacts" }

func (a artifactModel) toAPI() Artifact {
	return Artifact{
		ID:        a.ID,
		Kind:      a.Kind,
		SHA256:    a.SHA256,
		URL:       a.URL,
		Meta:      mapFromJSONMap(a.Meta),
		CreatedAt: a.CreatedAt,
	}
}

type factModel struct {
	ID        uuid.UUID         `gorm:"type:uuid;primaryKey"`
	MachineID uuid.UUID         `gorm:"type:uuid;not null"`
	Snapshot  datatypes.JSONMap `gorm:"type:jsonb"`
	CreatedAt time.Time         `gorm:"type:timestamptz;not null;default:now();autoCreateTime"`
}

func (factModel) TableName() string { return "facts" }

func (f factModel) toMap() map[string]any {
	return map[string]any{
		"id":         f.ID,
		"machine_id": f.MachineID,
		"snapshot":   mapFromJSONMap(f.Snapshot),
		"created_at": f.CreatedAt,
	}
}

func mapFromJSONMap(src datatypes.JSONMap) map[string]any {
	if src == nil {
		return map[string]any{}
	}
	out := make(map[string]any, len(src))
	for k, v := range src {
		out[k] = v
	}
	return out
}

func toJSONMap(src map[string]any) datatypes.JSONMap {
	out := datatypes.JSONMap{}
	if src == nil {
		return out
	}
	for k, v := range src {
		out[k] = v
	}
	return out
}

func valueOrZero(id *uuid.UUID) uuid.UUID {
	if id == nil {
		return uuid.Nil
	}
	return *id
}
