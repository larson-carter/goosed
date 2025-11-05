package migrations

import (
	"context"
	"database/sql"
	"time"

	"github.com/google/uuid"
	"github.com/pressly/goose/v3"
	"gorm.io/datatypes"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
	"gorm.io/gorm/schema"
)

func init() {
	goose.AddMigrationContext(upInit, downInit)
}

type Blueprint struct {
	ID        uuid.UUID         `gorm:"type:uuid;primaryKey"`
	Name      string            `gorm:"type:text;not null"`
	OS        string            `gorm:"type:text;not null"`
	Version   string            `gorm:"type:text;not null"`
	Data      datatypes.JSONMap `gorm:"type:jsonb"`
	CreatedAt time.Time         `gorm:"type:timestamptz;not null;default:now();autoCreateTime"`
	UpdatedAt time.Time         `gorm:"type:timestamptz;not null;default:now();autoUpdateTime"`
}

type Workflow struct {
	ID        uuid.UUID         `gorm:"type:uuid;primaryKey"`
	Name      string            `gorm:"type:text;not null"`
	Steps     datatypes.JSONMap `gorm:"type:jsonb"`
	CreatedAt time.Time         `gorm:"type:timestamptz;not null;default:now();autoCreateTime"`
	UpdatedAt time.Time         `gorm:"type:timestamptz;not null;default:now();autoUpdateTime"`
}

type Machine struct {
	ID        uuid.UUID         `gorm:"type:uuid;primaryKey"`
	MAC       string            `gorm:"type:text;uniqueIndex;not null"`
	Serial    string            `gorm:"type:text"`
	Profile   datatypes.JSONMap `gorm:"type:jsonb"`
	CreatedAt time.Time         `gorm:"type:timestamptz;not null;default:now();autoCreateTime"`
	UpdatedAt time.Time         `gorm:"type:timestamptz;not null;default:now();autoUpdateTime"`
}

type Run struct {
	ID          uuid.UUID  `gorm:"type:uuid;primaryKey"`
	MachineID   *uuid.UUID `gorm:"type:uuid"`
	BlueprintID *uuid.UUID `gorm:"type:uuid"`
	Status      string     `gorm:"type:text"`
	StartedAt   *time.Time `gorm:"type:timestamptz"`
	FinishedAt  *time.Time `gorm:"type:timestamptz"`
	Logs        string     `gorm:"type:text"`
	Machine     Machine    `gorm:"foreignKey:MachineID;references:ID;constraint:OnUpdate:CASCADE,OnDelete:SET NULL"`
	Blueprint   Blueprint  `gorm:"foreignKey:BlueprintID;references:ID;constraint:OnUpdate:CASCADE,OnDelete:SET NULL"`
}

type Artifact struct {
	ID        uuid.UUID         `gorm:"type:uuid;primaryKey"`
	Kind      string            `gorm:"type:text;not null"`
	SHA256    string            `gorm:"type:text;not null"`
	URL       string            `gorm:"type:text;not null"`
	Meta      datatypes.JSONMap `gorm:"type:jsonb"`
	CreatedAt time.Time         `gorm:"type:timestamptz;not null;default:now();autoCreateTime"`
}

type Fact struct {
	ID        uuid.UUID         `gorm:"type:uuid;primaryKey"`
	MachineID uuid.UUID         `gorm:"type:uuid;not null;index"`
	Snapshot  datatypes.JSONMap `gorm:"type:jsonb"`
	CreatedAt time.Time         `gorm:"type:timestamptz;not null;default:now();autoCreateTime"`
	Machine   Machine           `gorm:"foreignKey:MachineID;references:ID;constraint:OnUpdate:CASCADE,OnDelete:CASCADE"`
}

type Audit struct {
	ID      int64             `gorm:"type:bigserial;primaryKey"`
	Actor   string            `gorm:"type:text;not null"`
	Action  string            `gorm:"type:text;not null"`
	Obj     string            `gorm:"type:text"`
	Details datatypes.JSONMap `gorm:"type:jsonb"`
	At      time.Time         `gorm:"type:timestamptz;not null;default:now();autoCreateTime"`
}

func (Audit) TableName() string { return "audit" }

func upInit(ctx context.Context, tx *sql.Tx) error {
	gormDB, err := gorm.Open(postgres.New(postgres.Config{Conn: tx, PreferSimpleProtocol: true}), &gorm.Config{
		NamingStrategy: schema.NamingStrategy{SingularTable: false},
		Logger:         logger.Default.LogMode(logger.Silent),
	})
	if err != nil {
		return err
	}

	if err := gormDB.WithContext(ctx).AutoMigrate(
		&Blueprint{},
		&Workflow{},
		&Machine{},
		&Run{},
		&Artifact{},
		&Fact{},
		&Audit{},
	); err != nil {
		return err
	}

	m := gormDB.WithContext(ctx).Migrator()
	if err := m.CreateConstraint(&Run{}, "Machine"); err != nil {
		return err
	}
	if err := m.CreateConstraint(&Run{}, "Blueprint"); err != nil {
		return err
	}
	if err := m.CreateConstraint(&Fact{}, "Machine"); err != nil {
		return err
	}

	return nil
}

func downInit(ctx context.Context, tx *sql.Tx) error {
	gormDB, err := gorm.Open(postgres.New(postgres.Config{Conn: tx, PreferSimpleProtocol: true}), &gorm.Config{
		NamingStrategy: schema.NamingStrategy{SingularTable: false},
		Logger:         logger.Default.LogMode(logger.Silent),
	})
	if err != nil {
		return err
	}

	if err := gormDB.WithContext(ctx).Migrator().DropTable(
		&Audit{},
		&Fact{},
		&Artifact{},
		&Run{},
		&Machine{},
		&Workflow{},
		&Blueprint{},
	); err != nil {
		return err
	}

	return nil
}
