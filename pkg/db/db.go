package db

import (
	"context"
	"errors"
	"time"

	"github.com/georgysavva/scany/v2/pgxscan"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/pressly/goose/v3"

	_ "goosed/pkg/db/migrations"
)

const (
	// DefaultTimeout is used when executing queries to avoid leaking resources on hung calls.
	DefaultTimeout = 5 * time.Second
)

// Open creates a new pgx connection pool using the provided DSN.
func Open(ctx context.Context, dsn string) (*pgxpool.Pool, error) {
	cfg, err := pgxpool.ParseConfig(dsn)
	if err != nil {
		return nil, err
	}

	// Prefer simple protocol for compatibility with tools like goose.
	cfg.ConnConfig.DefaultQueryExecMode = pgx.QueryExecModeSimpleProtocol

	pool, err := pgxpool.NewWithConfig(ctx, cfg)
	if err != nil {
		return nil, err
	}

	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		return nil, err
	}

	return pool, nil
}

// Migrate runs all embedded SQL migrations against the provided pool.
func Migrate(ctx context.Context, pool *pgxpool.Pool) error {
	if pool == nil {
		return errors.New("nil pool provided")
	}

	if err := goose.SetDialect("postgres"); err != nil {
		return err
	}

	connString := pool.Config().ConnConfig.ConnString()
	sqlDB, err := goose.OpenDBWithDriver("pgx", connString)
	if err != nil {
		return err
	}
	defer sqlDB.Close()

	return goose.UpContext(ctx, sqlDB, "migrations")
}

// Exec executes a statement with the default timeout applied.
func Exec(ctx context.Context, pool *pgxpool.Pool, query string, args ...any) (pgconn.CommandTag, error) {
	ctx, cancel := context.WithTimeout(ctx, DefaultTimeout)
	defer cancel()

	return pool.Exec(ctx, query, args...)
}

// Get retrieves a single row into dest with the default timeout applied.
func Get(ctx context.Context, pool *pgxpool.Pool, dest any, query string, args ...any) error {
	ctx, cancel := context.WithTimeout(ctx, DefaultTimeout)
	defer cancel()

	return pgxscan.Get(ctx, pool, dest, query, args...)
}

// Select retrieves multiple rows into dest with the default timeout applied.
func Select(ctx context.Context, pool *pgxpool.Pool, dest any, query string, args ...any) error {
	ctx, cancel := context.WithTimeout(ctx, DefaultTimeout)
	defer cancel()

	return pgxscan.Select(ctx, pool, dest, query, args...)
}

// WithTimeout applies a custom timeout when executing operations using the provided function.
func WithTimeout(ctx context.Context, timeout time.Duration, fn func(context.Context) error) error {
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	return fn(ctx)
}

// Ping ensures the database is reachable with the default timeout.
func Ping(ctx context.Context, pool *pgxpool.Pool) error {
	ctx, cancel := context.WithTimeout(ctx, DefaultTimeout)
	defer cancel()
	return pool.Ping(ctx)
}
