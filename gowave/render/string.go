package render

import (
	"fmt"
	"github.com/ChenGuo505/gowave/internal/bytesconv"
	"net/http"
)

type String struct {
	Format string
	Data   []any
}

func (s *String) SetContentType(w http.ResponseWriter) {
	setContentType(w, "text/plain; charset=utf-8")
}

func (s *String) Render(w http.ResponseWriter) error {
	if len(s.Data) > 0 {
		_, err := fmt.Fprintf(w, s.Format, s.Data...)
		return err
	}
	_, err := w.Write(bytesconv.StringToBytes(s.Format))
	return err
}
