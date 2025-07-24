package render

import (
	"encoding/xml"
	"net/http"
)

type XML struct {
	Data any
}

func (x *XML) SetContentType(w http.ResponseWriter) {
	setContentType(w, "application/xml; charset=utf-8")
}

func (x *XML) Render(w http.ResponseWriter) error {
	return xml.NewEncoder(w).Encode(x.Data)
}
