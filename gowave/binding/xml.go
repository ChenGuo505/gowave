package binding

import (
	"encoding/xml"
	"net/http"
)

type xmlBinding struct {
}

func (x xmlBinding) Name() string {
	return "xml"
}

func (x xmlBinding) Bind(r *http.Request, obj any) error {
	if r.Body == nil {
		return nil
	}
	if err := xml.NewDecoder(r.Body).Decode(obj); err != nil {
		return err
	}
	return Validator.ValidateStruct(obj)
}
