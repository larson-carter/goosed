package api

import "encoding/json"

func (a *API) publishJSON(subject string, payload map[string]any) {
	if a.store.Bus == nil || subject == "" {
		return
	}
	data, err := json.Marshal(payload)
	if err != nil {
		return
	}
	_ = a.store.Bus.Publish(subject, data)
}
