package main

import (
	"context"
	"errors"
	"flag"
	"log"
	"os"
	"os/signal"
	"syscall"

	"goosed/services/agents/rhel"
)

func main() {
	configPath := flag.String("config", rhel.ConfigPath, "path to agent configuration file")
	flag.Parse()

	logger := log.New(os.Stdout, "goosed-agent-rhel: ", log.LstdFlags)

	svc, err := rhel.NewService(*configPath)
	if err != nil {
		logger.Fatalf("failed to initialize service: %v", err)
	}

	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer cancel()

	if err := svc.Run(ctx); err != nil && !errors.Is(err, context.Canceled) {
		logger.Fatalf("service exited with error: %v", err)
	}
}
