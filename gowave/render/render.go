package render

import "net/http"

type Render interface {
	SetContentType(w http.ResponseWriter)
	Render(w http.ResponseWriter) error
}

func setContentType(w http.ResponseWriter, value string) {
	w.Header().Set("Content-Type", value)
}
