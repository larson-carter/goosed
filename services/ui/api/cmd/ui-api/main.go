package main

import (
	"context"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/joho/godotenv"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"github.com/sethvargo/go-envconfig"

	"goosed/services/ui/api/internal/config"
	"goosed/services/ui/api/internal/db"
	"goosed/services/ui/api/internal/handlers"
	"goosed/services/ui/api/internal/otel"
	"goosed/services/ui/api/internal/version"
)

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	_ = godotenv.Load()

	zerolog.TimeFieldFormat = zerolog.TimeFormatUnix
	log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stderr})

	cfg, err := config.Load(ctx)
	if err != nil {
		log.Fatal().Err(err).Msg("load config")
	}

	if cfg.Addr == "" {
		cfg.Addr = ":8080"
	}

	cleanup, err := otel.Init(ctx, version.Name, cfg.OTLPEndpoint)
	if err != nil {
		log.Fatal().Err(err).Msg("init otel")
	}
	defer func() {
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := cleanup(shutdownCtx); err != nil {
			log.Error().Err(err).Msg("shutdown otel")
		}
	}()

	database, err := db.Connect(ctx, cfg.DBDSN)
	if err != nil {
		log.Fatal().Err(err).Msg("connect database")
	}
	defer func() {
		if err := db.Close(database); err != nil {
			log.Error().Err(err).Msg("close database")
		}
	}()

	if err := db.Migrate(ctx, database); err != nil {
		log.Fatal().Err(err).Msg("migrate database")
	}

	if err := db.Seed(ctx, database); err != nil {
		log.Fatal().Err(err).Msg("seed database")
	}

	r := handlers.Router(handlers.RouterOptions{AllowedOrigins: cfg.AllowedOrigins})

	srv := &http.Server{
		Addr:              cfg.Addr,
		Handler:           r,
		ReadHeaderTimeout: 10 * time.Second,
	}

	go func() {
		log.Info().Str("addr", cfg.Addr).Msg("starting goosed-ui-api")
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatal().Err(err).Msg("http server")
		}
	}()

	<-ctx.Done()

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := srv.Shutdown(shutdownCtx); err != nil {
		log.Error().Err(err).Msg("shutdown server")
	}
}

// compile-time check to ensure env tags remain valid when config changes
var _ = envconfig.Process
