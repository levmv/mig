package render

import (
	"html/template"
	"io"
	"io/fs"

	"github.com/levmv/mig"
)

// htmlRenderer implements the Renderer interface for html/template.
type htmlRenderer struct {
	Template *template.Template
}

// NewHTML creates and configures a renderer for html/template.
// It provides context-aware escaping, making it the safe choice for HTML output.
func NewHTML(fs fs.FS, funcMap FuncMap, patterns ...string) (mig.Renderer, error) {
	t, err := template.New("").Funcs(template.FuncMap(funcMap)).ParseFS(fs, patterns...)
	if err != nil {
		return nil, err
	}
	return &htmlRenderer{Template: t}, nil
}

// Render implements the mig.Renderer interface.
func (r *htmlRenderer) Render(w io.Writer, name string, data any) error {
	return r.Template.ExecuteTemplate(w, name, data)
}
