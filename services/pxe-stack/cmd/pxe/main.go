package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net"
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

	portsToTry := make([]int, 0, 1+len(cfg.HTTP.FallbackPorts))
	seenPorts := make(map[int]struct{}, 1+len(cfg.HTTP.FallbackPorts))
	if cfg.HTTP.Port > 0 {
		portsToTry = append(portsToTry, cfg.HTTP.Port)
		seenPorts[cfg.HTTP.Port] = struct{}{}
	}
	for _, port := range cfg.HTTP.FallbackPorts {
		if port <= 0 {
			continue
		}
		if _, exists := seenPorts[port]; exists {
			continue
		}
		portsToTry = append(portsToTry, port)
		seenPorts[port] = struct{}{}
	}
	if len(portsToTry) == 0 {
		portsToTry = append(portsToTry, 8080)
	}

	var (
		listener   net.Listener
		listenAddr string
	)
	preferredPort := portsToTry[0]
	for idx, port := range portsToTry {
		listenAddr = fmt.Sprintf(":%d", port)
		ln, err := listenWithRetry(ctx, listenAddr, 30*time.Second, 500*time.Millisecond, logger)
		if err != nil {
			if errors.Is(err, syscall.EADDRINUSE) && idx < len(portsToTry)-1 {
				if logger != nil {
					logger.Printf("WARN unable to bind HTTP listener on %s: %v; trying :%d next", listenAddr, err, portsToTry[idx+1])
				}
				continue
			}
			return fmt.Errorf("listen on %s: %w", listenAddr, err)
		}
		listener = ln
		if port != preferredPort && logger != nil {
			logger.Printf("INFO bound HTTP listener to fallback port :%d after preferred port :%d was unavailable", port, preferredPort)
		}
		cfg.HTTP.Port = port
		break
	}

	if listener == nil {
		return fmt.Errorf("failed to bind HTTP listener on any configured port")
	}

	server := &http.Server{
		Addr:    listener.Addr().String(),
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

	logger.Printf("INFO http listening on %s", listener.Addr())

	go func() {
		if err := server.Serve(listener); err != nil && !errors.Is(err, http.ErrServerClosed) {
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

func listenWithRetry(ctx context.Context, address string, maxWait time.Duration, initialDelay time.Duration, logger *log.Logger) (net.Listener, error) {
	if maxWait <= 0 {
		maxWait = 30 * time.Second
	}
	if initialDelay <= 0 {
		initialDelay = 100 * time.Millisecond
	}

	deadline := time.Now().Add(maxWait)
	delay := initialDelay
	const maxDelay = 5 * time.Second

	for {
		ln, err := net.Listen("tcp", address)
		if err == nil {
			return ln, nil
		}

		if !errors.Is(err, syscall.EADDRINUSE) {
			return nil, err
		}

		remaining := time.Until(deadline)
		if remaining <= 0 {
			return nil, fmt.Errorf("timed out waiting for %s to become available: %w", address, err)
		}

		sleep := delay
		if sleep > remaining {
			sleep = remaining
		}

		if logger != nil {
			logger.Printf("WARN retrying listen on %s after error: %v (waiting %s before next attempt)", address, err, sleep)
		}

		timer := time.NewTimer(sleep)
		select {
		case <-ctx.Done():
			if !timer.Stop() {
				<-timer.C
			}
			return nil, fmt.Errorf("context canceled while waiting for %s: %w", address, ctx.Err())
		case <-timer.C:
		}

		delay *= 2
		if delay > maxDelay {
			delay = maxDelay
		}
	}
}
