package render

import (
	"github.com/ChenGuo505/gowave/internal/bytesconv"
	"html/template"
	"net/http"
)

type HTML struct {
	Data     any
	Name     string
	Template *template.Template

	IsTemplate bool
}

func (h *HTML) SetContentType(w http.ResponseWriter) {
	setContentType(w, "text/html; charset=utf-8")
}

func (h *HTML) Render(w http.ResponseWriter) error {
	if h.IsTemplate {
		return h.Template.ExecuteTemplate(w, h.Name, h.Data)
	}
	_, err := w.Write(bytesconv.StringToBytes(h.Data.(string)))
	return err
}

type HTMLRender struct {
	Template *template.Template
}
