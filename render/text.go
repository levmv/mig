package render

import (
	"io"
	"io/fs"
	"text/template"

	"github.com/levmv/mig"
)

// textRenderer implements the Renderer interface for text/template.
type textRenderer struct {
	Template *template.Template
}

// NewText creates and configures a renderer for text/template.
// Use this for generating non-HTML content like emails, reports, or configuration files.
func NewText(fs fs.FS, funcMap FuncMap, patterns ...string) (mig.Renderer, error) {
	t, err := template.New("").Funcs(template.FuncMap(funcMap)).ParseFS(fs, patterns...)
	if err != nil {
		return nil, err
	}
	return &textRenderer{Template: t}, nil
}

// Render implements the mig.Renderer interface.
func (r *textRenderer) Render(w io.Writer, name string, data any) error {
	return r.Template.ExecuteTemplate(w, name, data)
}
