package gowave

import (
	"encoding/json"
	"encoding/xml"
	"fmt"
	"html/template"
	"net/http"
	"net/url"
)

type Context struct {
	W   http.ResponseWriter
	Req *http.Request

	engine *Engine
}

func (c *Context) HTML(code int, html string) error {
	c.W.Header().Set("Content-Type", "text/html; charset=utf-8")
	c.W.WriteHeader(code)
	_, err := c.W.Write(StringToBytes(html))
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

func (c *Context) Template(name string, data any) error {
	c.W.Header().Set("Content-Type", "text/html; charset=utf-8")
	err := c.engine.HTMLRender.Template.ExecuteTemplate(c.W, name, data)
	return err
}

func (c *Context) JSON(code int, data any) error {
	c.W.Header().Set("Content-Type", "application/json; charset=utf-8")
	c.W.WriteHeader(code)
	// Assuming a JSON encoding function is available
	jsonData, err := json.Marshal(data)
	if err != nil {
		return err
	}
	_, err = c.W.Write(jsonData)
	return err
}

func (c *Context) XML(code int, data any) error {
	c.W.Header().Set("Content-Type", "application/xml; charset=utf-8")
	c.W.WriteHeader(code)
	err := xml.NewEncoder(c.W).Encode(data)
	return err
}

func (c *Context) File(filename string) {
	http.ServeFile(c.W, c.Req, filename)
}

func (c *Context) FileAttachment(filepath, filename string) {
	if IsASCII(filename) {
		c.W.Header().Set("Content-Disposition", "attachment; filename="+filename)
	} else {
		c.W.Header().Set("Content-Disposition", "attachment; filename*=UTF-8''"+url.QueryEscape(filename))
	}
	http.ServeFile(c.W, c.Req, filepath)
}

func (c *Context) FileFromFS(filepath string, fs http.FileSystem) {
	defer func(old string) {
		c.Req.URL.Path = old
	}(c.Req.URL.Path)
	c.Req.URL.Path = filepath
	http.FileServer(fs).ServeHTTP(c.W, c.Req)
}

func (c *Context) Redirect(code int, url string) {
	if (code < http.StatusMultipleChoices || code >= http.StatusPermanentRedirect) && code != http.StatusCreated {
		panic(fmt.Sprintf("invalid redirect code: %d", code))
	}
	http.Redirect(c.W, c.Req, url, code)
}

func (c *Context) String(code int, format string, args ...any) error {
	c.W.Header().Set("Content-Type", "text/plain; charset=utf-8")
	c.W.WriteHeader(code)
	if len(args) > 0 {
		_, err := fmt.Fprintf(c.W, format, args...)
		return err
	}
	_, err := c.W.Write(StringToBytes(format))
	return err
}
