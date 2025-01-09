package mig

import (
	"io"
	"io/fs"
	"text/template"
)

type Renderer interface {
	Render(io.Writer, string, any) error
}

type TemplateRenderer struct {
	Template *template.Template
}

func (h *TemplateRenderer) Render(wr io.Writer, name string, data any) error {
	return h.Template.ExecuteTemplate(wr, name, data)
}

func (h *TemplateRenderer) Funcs(fncs template.FuncMap) {
	h.Template.Funcs(fncs)
}

func NewTemplateRenderer(tfs fs.FS, patterns ...string) (*TemplateRenderer, error) {
	t, err := template.ParseFS(tfs, patterns...)
	if err != nil {
		return nil, err
	}

	return &TemplateRenderer{
		Template: t,
	}, nil
}

type DynamicTemplateRenderer struct {
	template *template.Template
}

func (h *DynamicTemplateRenderer) Render(wr io.Writer, name string, data any) error {
	return h.template.ExecuteTemplate(wr, name, data)
}

func NewDynamicTemplateRenderer(tfs fs.FS, patterns ...string) (Renderer, error) {
	t, err := template.ParseFS(tfs, patterns...)
	if err != nil {
		return nil, err
	}
	return &TemplateRenderer{
		Template: t,
	}, nil
}
