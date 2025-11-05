package api

import (
	"errors"
	"fmt"
	"net/http"
	"strings"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

func (a *API) handleIPXE(w http.ResponseWriter, r *http.Request) {
	mac := strings.ToLower(strings.TrimSpace(r.URL.Query().Get("mac")))
	if mac == "" {
		respondError(w, http.StatusBadRequest, errors.New("mac query parameter is required"))
		return
	}

	machine, err := a.fetchMachineByMAC(r.Context(), mac)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			respondError(w, http.StatusNotFound, fmt.Errorf("machine with mac %s not found", mac))
			return
		}
		respondError(w, http.StatusInternalServerError, err)
		return
	}

	token, ok, err := a.tokens.Active(r.Context(), machine.MAC)
	if err != nil {
		respondError(w, http.StatusInternalServerError, err)
		return
	}
	if !ok {
		token, err = a.tokens.Issue(r.Context(), machine.MAC)
		if err != nil {
			respondError(w, http.StatusInternalServerError, err)
			return
		}
	}

	if err := a.tokens.MarkUsed(r.Context(), token.Value); err != nil {
		respondError(w, http.StatusInternalServerError, err)
		return
	}

	apiBase := a.config.APIBase
	if apiBase == "" {
		scheme := "http"
		if r.TLS != nil {
			scheme = "https"
		}
		apiBase = fmt.Sprintf("%s://%s", scheme, r.Host)
	}

	payload := map[string]any{
		"Token":   token.Value,
		"MAC":     machine.MAC,
		"APIBase": apiBase,
	}

	rendered, err := a.renderer.Render("ipxe.tmpl", payload)
	if err != nil {
		respondError(w, http.StatusInternalServerError, err)
		return
	}

	w.Header().Set("Content-Type", "text/plain")
	_, _ = w.Write([]byte(rendered))
}

func (a *API) handleKickstart(w http.ResponseWriter, r *http.Request) {
	machineID, err := uuid.Parse(strings.TrimSpace(r.URL.Query().Get("machine_id")))
	if err != nil {
		respondError(w, http.StatusBadRequest, errors.New("valid machine_id is required"))
		return
	}

	machine, err := a.fetchMachineByID(r.Context(), machineID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			respondError(w, http.StatusNotFound, fmt.Errorf("machine %s not found", machineID))
			return
		}
		respondError(w, http.StatusInternalServerError, err)
		return
	}

	issuedToken, err := a.tokens.Issue(r.Context(), machine.MAC)
	if err != nil {
		respondError(w, http.StatusInternalServerError, err)
		return
	}

	apiBase := a.config.APIBase
	if apiBase == "" {
		scheme := "http"
		if r.TLS != nil {
			scheme = "https"
		}
		apiBase = fmt.Sprintf("%s://%s", scheme, r.Host)
	}

	rendered, err := a.renderer.Render("kickstart.tmpl", map[string]any{
		"Machine": machine,
		"Profile": machine.Profile,
		"Token":   issuedToken.Value,
		"APIBase": apiBase,
	})
	if err != nil {
		respondError(w, http.StatusInternalServerError, err)
		return
	}

	w.Header().Set("Content-Type", "text/plain")
	_, _ = w.Write([]byte(rendered))
}

func (a *API) handleUnattend(w http.ResponseWriter, r *http.Request) {
	machineID, err := uuid.Parse(strings.TrimSpace(r.URL.Query().Get("machine_id")))
	if err != nil {
		respondError(w, http.StatusBadRequest, errors.New("valid machine_id is required"))
		return
	}

	machine, err := a.fetchMachineByID(r.Context(), machineID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			respondError(w, http.StatusNotFound, fmt.Errorf("machine %s not found", machineID))
			return
		}
		respondError(w, http.StatusInternalServerError, err)
		return
	}

	issuedToken, err := a.tokens.Issue(r.Context(), machine.MAC)
	if err != nil {
		respondError(w, http.StatusInternalServerError, err)
		return
	}

	apiBase := a.config.APIBase
	if apiBase == "" {
		scheme := "http"
		if r.TLS != nil {
			scheme = "https"
		}
		apiBase = fmt.Sprintf("%s://%s", scheme, r.Host)
	}

	rendered, err := a.renderer.Render("unattend.xml.tmpl", map[string]any{
		"Machine": machine,
		"Profile": machine.Profile,
		"Token":   issuedToken.Value,
		"APIBase": apiBase,
	})
	if err != nil {
		respondError(w, http.StatusInternalServerError, err)
		return
	}

	w.Header().Set("Content-Type", "application/xml")
	_, _ = w.Write([]byte(rendered))
}
