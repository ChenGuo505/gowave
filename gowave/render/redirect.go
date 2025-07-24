package render

import (
	"fmt"
	"net/http"
)

type Redirect struct {
	Code int
	Req  *http.Request
	URL  string
}

func (r *Redirect) SetContentType(w http.ResponseWriter) {
	// Redirects do not require a specific content type, but we can set it to text/plain
	// to avoid any issues with clients that expect a content type.
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
}

func (r *Redirect) Render(w http.ResponseWriter) error {
	if (r.Code < http.StatusMultipleChoices || r.Code >= http.StatusPermanentRedirect) && r.Code != http.StatusCreated {
		return fmt.Errorf("invalid redirect status code: %d", r.Code)
	}
	http.Redirect(w, r.Req, r.URL, r.Code)
	return nil
}
