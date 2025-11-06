package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"sync/atomic"
	"syscall"
	"time"

	"github.com/prometheus/client_golang/prometheus/promhttp"

	"goosed/pkg/telemetry"
	"goosed/services/pxe-stack/internal/config"
	"goosed/services/pxe-stack/internal/dhcp"
	"goosed/services/pxe-stack/internal/pxehttp"
	"goosed/services/pxe-stack/internal/tftp"
)

func main() {
	if err := run("pxe-stack"); err != nil {
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

	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("load config: %w", err)
	}

	var dhcpReady, tftpReady, httpReady atomic.Bool

	errCh := make(chan error, 3)

	if cfg.DHCP.Enabled {
		server, err := dhcp.NewServer(cfg.DHCP, logger)
		if err != nil {
			return fmt.Errorf("create dhcp server: %w", err)
		}
		go func() {
			if err := server.Run(ctx, &dhcpReady); err != nil {
				errCh <- fmt.Errorf("dhcp: %w", err)
			}
		}()
	} else {
		dhcpReady.Store(true)
	}

	if cfg.TFTP.Enabled {
		server := tftp.NewServer(cfg.TFTP, logger)
		go func() {
			if err := server.Run(ctx, &tftpReady); err != nil {
				errCh <- fmt.Errorf("tftp: %w", err)
			}
		}()
	} else {
		tftpReady.Store(true)
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/healthz", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
	mux.HandleFunc("/readyz", func(w http.ResponseWriter, r *http.Request) {
		if dhcpReady.Load() && tftpReady.Load() && httpReady.Load() {
			w.WriteHeader(http.StatusOK)
			return
		}
		http.Error(w, "components not ready", http.StatusServiceUnavailable)
	})
	mux.Handle("/metrics", promhttp.Handler())

	if cfg.HTTP.Enabled {
		if err := pxehttp.RegisterHandlers(mux, cfg.HTTP, &httpReady, logger); err != nil {
			return fmt.Errorf("register http handlers: %w", err)
		}
	} else {
		httpReady.Store(true)
	}

	server := &http.Server{
		Addr:    fmt.Sprintf(":%d", cfg.HTTP.Port),
		Handler: middleware(mux),
	}

	go func() {
		<-ctx.Done()
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		if err := server.Shutdown(shutdownCtx); err != nil && !errors.Is(err, http.ErrServerClosed) {
			fmt.Fprintf(os.Stderr, "%s: http shutdown error: %v\n", serviceName, err)
		}
	}()

	logger.Printf("INFO http listening on %s", server.Addr)

	go func() {
		if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			errCh <- fmt.Errorf("http: %w", err)
		}
	}()

	select {
	case err := <-errCh:
		return err
	case <-ctx.Done():
		return nil
	}
}
