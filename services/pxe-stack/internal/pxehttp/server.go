package pxehttp

import (
	"errors"
	"fmt"
	"io/fs"
	"log"
	"net/http"
	"os"
	"strings"
	"sync/atomic"

	brandingfs "goosed/infra/branding"
	"goosed/services/pxe-stack/internal/config"
)

func RegisterHandlers(mux *http.ServeMux, cfg config.HTTPConfig, ready *atomic.Bool, logger *log.Logger) error {
	if mux == nil {
		return errors.New("nil mux")
	}
	if logger == nil {
		logger = log.Default()
	}
	if ready == nil {
		return errors.New("ready indicator is nil")
	}
	if cfg.APIEndpoint == "" {
		return errors.New("PXE HTTP API endpoint is required")
	}

	mux.HandleFunc("/menu.ipxe", func(w http.ResponseWriter, r *http.Request) {
		mac := strings.TrimSpace(r.URL.Query().Get("mac"))
		if mac == "" {
			http.Error(w, "missing mac query parameter", http.StatusBadRequest)
			return
		}
		script := fmt.Sprintf("#!ipxe\nset api %s\nchain ${api}/v1/boot/ipxe?mac=${mac}\n", cfg.APIEndpoint)
		w.Header().Set("Content-Type", "text/plain")
		_, _ = w.Write([]byte(script))
	})

	fileSystem, err := brandingFileSystem(cfg.BrandingFS)
	if err != nil {
		return fmt.Errorf("prepare branding filesystem: %w", err)
	}
	mux.Handle("/branding/", http.StripPrefix("/branding/", http.FileServer(http.FS(fileSystem))))

	ready.Store(true)
	logger.Printf("INFO http PXE handlers registered for API %s", cfg.APIEndpoint)
	return nil
}

func brandingFileSystem(path string) (fs.FS, error) {
	if path == "" {
		return fs.Sub(brandingfs.Files, ".")
	}
	info, err := os.Stat(path)
	if err != nil {
		return nil, err
	}
	if !info.IsDir() {
		return nil, fmt.Errorf("branding path %s is not a directory", path)
	}
	return os.DirFS(path), nil
}
