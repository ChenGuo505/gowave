package binding

import (
	"encoding/json"
	"errors"
	"net/http"
)

type jsonBinding struct {
	DisallowUnknownFields bool // Disallow unknown fields in JSON parsing
	EnableJsonValidation  bool // Enable JSON validation
}

func (j *jsonBinding) Name() string {
	return "json"
}

func (j *jsonBinding) Bind(r *http.Request, obj any) error {
	body := r.Body
	if body == nil {
		return errors.New("request body is nil")
	}
	decoder := json.NewDecoder(body)
	if j.DisallowUnknownFields {
		decoder.DisallowUnknownFields()
	}
	err := decoder.Decode(obj)
	if err != nil {
		return err
	}
	if j.EnableJsonValidation {
		return Validator.ValidateStruct(obj)
	}
	return nil
}
