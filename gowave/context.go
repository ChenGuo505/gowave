package gowave

import (
	"html/template"
	"net/http"
)

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

func (c *Context) HTMLTemplate(name string, data any, filenames ...string) error {
	c.W.Header().Set("Content-Type", "text/html; charset=utf-8")
	t := template.New(name)
	t, err := t.ParseFiles(filenames...)
	if err != nil {
		return err
	}
	if err := t.Execute(c.W, data); err != nil {
		return err
	}
	return nil
}

func (c *Context) HTMLTemplateGlob(name string, data any, pattern string) error {
	c.W.Header().Set("Content-Type", "text/html; charset=utf-8")
	t := template.New(name)
	t, err := t.ParseGlob(pattern)
	if err != nil {
		return err
	}
	if err := t.Execute(c.W, data); err != nil {
		return err
	}
	return nil
}
