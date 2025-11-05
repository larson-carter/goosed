package rhel

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"
)

const (
	// ConfigPath is where the agent expects to find its JSON configuration file.
	ConfigPath = "/etc/goosed/agent.conf"

	defaultStateDir       = "/var/lib/goosed"
	postinstallMarkerName = "postinstall_done"
)

// Config represents the agent configuration stored on disk.
type Config struct {
	API       string `json:"api"`
	Token     string `json:"token"`
	MachineID string `json:"machine_id"`
}

// Service is the long-running background process that periodically posts
// machine facts back to the Goosed API.
type Service struct {
	client   *http.Client
	config   Config
	logger   *log.Logger
	stateDir string
}

// NewService loads configuration from the provided path and returns an
// initialized Service instance.
func NewService(configPath string) (*Service, error) {
	cfg, err := loadConfig(configPath)
	if err != nil {
		return nil, err
	}

	if strings.TrimSpace(cfg.API) == "" {
		return nil, fmt.Errorf("config missing api field")
	}

	if err := ensureHTTPS(cfg.API, allowInsecureHTTP()); err != nil {
		return nil, err
	}

	if strings.TrimSpace(cfg.MachineID) == "" {
		return nil, fmt.Errorf("config missing machine_id field")
	}

	logger := log.New(os.Stdout, "goosed-agent-rhel: ", log.LstdFlags)

	svc := &Service{
		client:   &http.Client{Timeout: 15 * time.Second},
		config:   cfg,
		logger:   logger,
		stateDir: defaultStateDir,
	}

	return svc, nil
}

// Run executes the agent loop until the provided context is cancelled.
func (s *Service) Run(ctx context.Context) error {
	if err := s.reportOnce(ctx); err != nil {
		s.logger.Printf("initial report failed: %v", err)
	}

	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
			if err := s.reportOnce(ctx); err != nil {
				s.logger.Printf("report failed: %v", err)
			}
		}
	}
}

func (s *Service) reportOnce(ctx context.Context) error {
	snapshot, firstRun, err := s.buildSnapshot()
	if err != nil {
		return fmt.Errorf("build snapshot: %w", err)
	}

	if err := s.sendFacts(ctx, snapshot); err != nil {
		return err
	}

	if firstRun {
		if err := s.markPostinstallComplete(); err != nil {
			s.logger.Printf("failed to mark postinstall completion: %v", err)
		}
	}

	return nil
}

func (s *Service) buildSnapshot() (map[string]any, bool, error) {
	firstRun := !s.postinstallMarkerExists()

	kernel, err := readKernelRelease()
	if err != nil {
		return nil, firstRun, fmt.Errorf("read kernel release: %w", err)
	}

	selinuxStatus := readSELinuxStatus()

	snapshot := map[string]any{
		"kernel":  kernel,
		"selinux": selinuxStatus,
	}

	if firstRun {
		snapshot["packages"] = []string{"placeholder"}
		snapshot["postinstall_done"] = true
	}

	return snapshot, firstRun, nil
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

func (s *Service) postinstallMarkerExists() bool {
	marker := filepath.Join(s.stateDir, postinstallMarkerName)
	info, err := os.Stat(marker)
	if err != nil {
		return false
	}
	return !info.IsDir()
}

func (s *Service) markPostinstallComplete() error {
	if err := os.MkdirAll(s.stateDir, 0o755); err != nil {
		return fmt.Errorf("create state dir: %w", err)
	}

	marker := filepath.Join(s.stateDir, postinstallMarkerName)
	if err := os.WriteFile(marker, []byte(time.Now().Format(time.RFC3339)), 0o644); err != nil {
		return fmt.Errorf("write marker: %w", err)
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

	return cfg, nil
}

func readKernelRelease() (string, error) {
	data, err := os.ReadFile("/proc/sys/kernel/osrelease")
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(data)), nil
}

func readSELinuxStatus() string {
	data, err := os.ReadFile("/sys/fs/selinux/enforce")
	if err != nil {
		return "disabled"
	}

	value := strings.TrimSpace(string(data))
	switch value {
	case "1":
		return "enforcing"
	case "0":
		return "permissive"
	default:
		return value
	}
}

func allowInsecureHTTP() bool {
	value := strings.ToLower(strings.TrimSpace(os.Getenv("GOOSED_ALLOW_INSECURE_HTTP")))
	switch value {
	case "1", "true", "yes", "on":
		return true
	default:
		return false
	}
}

func ensureHTTPS(raw string, allowInsecure bool) error {
	parsed, err := url.Parse(raw)
	if err != nil {
		return fmt.Errorf("parse api url: %w", err)
	}

	switch parsed.Scheme {
	case "https":
		return nil
	case "http", "":
		if allowInsecure {
			return nil
		}
		if parsed.Scheme == "" {
			return fmt.Errorf("api url must include https scheme")
		}
		return fmt.Errorf("api url must use https: %s", raw)
	default:
		if allowInsecure {
			return nil
		}
		return fmt.Errorf("api url must use https: %s", raw)
	}
}
