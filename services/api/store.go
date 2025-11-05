package api

import (
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/nats-io/nats.go"
	"gorm.io/gorm"

	gos3 "goosed/pkg/s3"
)

// Store holds external dependencies required by the API layer.
type Store struct {
	DB  *pgxpool.Pool
	ORM *gorm.DB
	S3  *gos3.Client
	Bus *nats.Conn
}
