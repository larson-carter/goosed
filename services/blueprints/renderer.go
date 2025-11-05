package blueprints

import (
	"sync"

	"goosed/pkg/render"
)

var (
	rendererOnce sync.Once
	renderer     *render.Engine
	rendererErr  error
)

func getRenderer() (*render.Engine, error) {
	rendererOnce.Do(func() {
		renderer, rendererErr = render.New()
	})
	return renderer, rendererErr
}

// RenderKickstart renders the Kickstart template using the provided profile data.
func RenderKickstart(profile map[string]any) string {
	return renderProfileTemplate("kickstart.tmpl", profile)
}

// RenderUnattend renders the Windows unattend XML template using the provided profile data.
func RenderUnattend(profile map[string]any) string {
	return renderProfileTemplate("unattend.xml.tmpl", profile)
}

func renderProfileTemplate(name string, profile map[string]any) string {
	if profile == nil {
		profile = map[string]any{}
	}

	engine, err := getRenderer()
	if err != nil {
		return ""
	}

	payload := map[string]any{
		"Machine": map[string]any{
			"ID":     "",
			"MAC":    "",
			"Serial": "",
		},
		"Profile": profile,
	}

	rendered, err := engine.Render(name, payload)
	if err != nil {
		return ""
	}

	return rendered
}
