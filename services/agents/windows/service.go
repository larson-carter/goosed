//go:build windows

package windows

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"golang.org/x/sys/windows/svc"
)

const (
	// ConfigPath is where the Windows agent expects its configuration file.
	ConfigPath = `C:\\ProgramData\\Goosed\\agent.conf`

	serviceName = "GoosedAgent"
)

// Config represents the Windows agent configuration stored on disk.
type Config struct {
	API       string `json:"api"`
	Token     string `json:"token"`
	MachineID string `json:"machine_id"`
}

// Service posts machine facts back to the Goosed API.
type Service struct {
	client *http.Client
	config Config
	logger *log.Logger
}

// Run initialises the Windows service entrypoint. When executed interactively it
// posts a single snapshot and exits. When invoked by the Service Control Manager
// it posts the snapshot and then waits to be stopped.
func Run() error {
	cfg, err := loadConfig(ConfigPath)
	if err != nil {
		return err
	}

	logger := log.New(os.Stdout, "goosed-agent-windows: ", log.LstdFlags)
	svcInstance := &Service{
		client: &http.Client{Timeout: 15 * time.Second},
		config: cfg,
		logger: logger,
	}

	isService, err := svc.IsWindowsService()
	if err != nil {
		return fmt.Errorf("detecting service environment: %w", err)
	}

	if !isService {
		return svcInstance.RunOnce(context.Background())
	}

	return svc.Run(serviceName, &program{svc: svcInstance})
}

// RunOnce posts a heartbeat snapshot to the API.
func (s *Service) RunOnce(ctx context.Context) error {
	snapshot := map[string]any{
		"platform":  "windows",
		"heartbeat": time.Now().UTC().Format(time.RFC3339),
	}

	if err := s.sendFacts(ctx, snapshot); err != nil {
		return err
	}

	s.logger.Printf("posted heartbeat for machine %s", s.config.MachineID)
	return nil
}

func (s *Service) sendFacts(ctx context.Context, snapshot map[string]any) error {
	payload := map[string]any{
		"machine_id": s.config.MachineID,
		"snapshot":   snapshot,
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("marshal payload: %w", err)
	}

	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	url := strings.TrimRight(s.config.API, "/") + "/v1/agents/facts"

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	if token := strings.TrimSpace(s.config.Token); token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}

	resp, err := s.client.Do(req)
	if err != nil {
		if errors.Is(err, context.Canceled) {
			return err
		}
		return fmt.Errorf("post facts: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		data, _ := io.ReadAll(io.LimitReader(resp.Body, 2048))
		return fmt.Errorf("post facts unexpected status %d: %s", resp.StatusCode, strings.TrimSpace(string(data)))
	}

	if _, err := io.Copy(io.Discard, resp.Body); err != nil {
		return fmt.Errorf("drain response body: %w", err)
	}

	return nil
}

func loadConfig(path string) (Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return Config{}, fmt.Errorf("read config: %w", err)
	}

	var cfg Config
	if err := json.Unmarshal(data, &cfg); err != nil {
		return Config{}, fmt.Errorf("parse config: %w", err)
	}

	if strings.TrimSpace(cfg.API) == "" {
		return Config{}, fmt.Errorf("config missing api field")
	}

	if strings.TrimSpace(cfg.MachineID) == "" {
		return Config{}, fmt.Errorf("config missing machine_id field")
	}

	return cfg, nil
}

type program struct {
	svc *Service
}

func (p *program) Execute(_ []string, r <-chan svc.ChangeRequest, changes chan<- svc.Status) (bool, uint32) {
	const accepted = svc.AcceptStop | svc.AcceptShutdown
	changes <- svc.Status{State: svc.StartPending}
	changes <- svc.Status{State: svc.Running, Accepts: accepted}

	if err := p.svc.RunOnce(context.Background()); err != nil && !errors.Is(err, context.Canceled) {
		p.svc.logger.Printf("failed to post heartbeat: %v", err)
	}

	for c := range r {
		switch c.Cmd {
		case svc.Interrogate:
			changes <- c.CurrentStatus
		case svc.Stop, svc.Shutdown:
			changes <- svc.Status{State: svc.StopPending}
			return false, 0
		default:
		}
	}

	changes <- svc.Status{State: svc.StopPending}
	return false, 0
}
