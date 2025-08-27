package gateway

import "net/http"

type Config struct {
	Name      string
	Path      string
	SetHeader func(req *http.Request)
}
