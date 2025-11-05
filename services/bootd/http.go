package bootd

import (
	"errors"
	"fmt"
	"io/fs"
	"net/http"
	"strings"

	brandingfs "goosed/infra/branding"
)

const apiEndpoint = "http://api.goose.local"

// RegisterHandlers wires HTTP handlers for PXE boot helpers and static assets.
func RegisterHandlers(mux *http.ServeMux) error {
	if mux == nil {
		return errors.New("nil mux")
	}

	mux.HandleFunc("/menu.ipxe", menuHandler)

	sub, err := fs.Sub(brandingfs.Files, ".")
	if err != nil {
		return fmt.Errorf("prepare branding fs: %w", err)
	}

	fileServer := http.FileServer(http.FS(sub))
	mux.Handle("/branding/", http.StripPrefix("/branding/", fileServer))

	return nil
}

func menuHandler(w http.ResponseWriter, r *http.Request) {
	mac := strings.TrimSpace(r.URL.Query().Get("mac"))
	if mac == "" {
		http.Error(w, "missing mac query parameter", http.StatusBadRequest)
		return
	}

	script := fmt.Sprintf("#!ipxe\nset api %s\nchain ${api}/v1/boot/ipxe?mac=${mac}\n", apiEndpoint)

	w.Header().Set("Content-Type", "text/plain")
	_, _ = w.Write([]byte(script))
}
