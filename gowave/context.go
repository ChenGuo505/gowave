package gowave

import "net/http"

type Context struct {
	W   http.ResponseWriter
	Req *http.Request
}

func (c *Context) HTML(code int, html string) error {
	c.W.Header().Set("Content-Type", "text/html; charset=utf-8")
	c.W.WriteHeader(code)
	_, err := c.W.Write([]byte(html))
	return err
}
