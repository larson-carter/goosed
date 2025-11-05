package render

import (
	"bytes"
	"embed"
	"fmt"
	"text/template"
)

//go:embed templates/*.tmpl
var templatesFS embed.FS

// Engine renders templates embedded in the package.
type Engine struct {
	templates *template.Template
}

// New initialises an Engine by parsing all embedded templates.
func New() (*Engine, error) {
	t, err := template.New("render").Funcs(template.FuncMap{}).ParseFS(templatesFS, "templates/*.tmpl")
	if err != nil {
		return nil, fmt.Errorf("parse templates: %w", err)
	}
	return &Engine{templates: t}, nil
}

// Render executes the named template with the provided data and returns the rendered string.
func (e *Engine) Render(name string, data any) (string, error) {
	if e == nil || e.templates == nil {
		return "", fmt.Errorf("nil engine")
	}

	buf := bytes.NewBuffer(nil)
	if err := e.templates.ExecuteTemplate(buf, name, data); err != nil {
		return "", err
	}

	return buf.String(), nil
}
