package api

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/datatypes"
)

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
