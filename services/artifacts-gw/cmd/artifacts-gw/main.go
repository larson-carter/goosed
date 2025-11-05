package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/prometheus/client_golang/prometheus/promhttp"

	gos3 "goosed/pkg/s3"
	"goosed/pkg/telemetry"
	"goosed/services/artifacts-gw"
)

func main() {
	if err := run("artifacts-gw"); err != nil {
		log.New(os.Stderr, "", log.LstdFlags).Fatal(err)
	}
}

func run(serviceName string) error {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	shutdownTelemetry, middleware, logger, err := telemetry.Init(ctx, serviceName)
	if err != nil {
		return fmt.Errorf("init telemetry: %w", err)
	}

	defer func() {
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if shutdownTelemetry != nil {
			if err := shutdownTelemetry(shutdownCtx); err != nil {
				fmt.Fprintf(os.Stderr, "%s: telemetry shutdown error: %v\n", serviceName, err)
			}
		}
	}()

	s3Client, err := gos3.NewClientFromEnv()
	if err != nil {
		return fmt.Errorf("init s3 client: %w", err)
	}

	bucket := strings.TrimSpace(os.Getenv("S3_BUCKET"))
	if bucket == "" {
		return errors.New("S3_BUCKET is required")
	}

	presignServer, err := artifactsgw.NewServer(bucket, s3Client)
	if err != nil {
		return fmt.Errorf("init presign server: %w", err)
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/healthz", healthHandler)
	mux.HandleFunc("/readyz", readyHandler)
	mux.Handle("/metrics", promhttp.Handler())

	if err := presignServer.RegisterHandlers(mux); err != nil {
		return fmt.Errorf("register presign handlers: %w", err)
	}

	server := &http.Server{
		Addr:    ":8080",
		Handler: middleware(mux),
	}

	go func() {
		<-ctx.Done()
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		if err := server.Shutdown(shutdownCtx); err != nil && !errors.Is(err, http.ErrServerClosed) {
			fmt.Fprintf(os.Stderr, "%s: server shutdown error: %v\n", serviceName, err)
		}
	}()

	logger.Printf("INFO listening on %s", server.Addr)

	if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		logger.Printf("ERROR server failed: %v", err)
		return err
	}

	return nil
}

func healthHandler(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
}

func readyHandler(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
}
