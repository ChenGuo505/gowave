package render

import (
	"encoding/json"
	"net/http"
)

type JSON struct {
	Data any
}

func (j *JSON) SetContentType(w http.ResponseWriter) {
	setContentType(w, "application/json; charset=utf-8")
}

func (j *JSON) Render(w http.ResponseWriter) error {
	jsonData, err := json.Marshal(j.Data)
	if err != nil {
		return err
	}
	_, err = w.Write(jsonData)
	return err
}
