package gowave

import (
	"github.com/ChenGuo505/gowave/render"
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
	return c.Render(code, &render.HTML{Data: html, IsTemplate: false})
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
	return c.Render(http.StatusOK, &render.HTML{
		Data:       data,
		Name:       name,
		IsTemplate: true,
		Template:   c.engine.HTMLRender.Template,
	})
}

func (c *Context) JSON(code int, data any) error {
	return c.Render(code, &render.JSON{Data: data})
}

func (c *Context) XML(code int, data any) error {
	return c.Render(code, &render.XML{Data: data})
}

func (c *Context) Redirect(code int, url string) error {
	return c.Render(code, &render.Redirect{Code: code, Req: c.Req, URL: url})
}

func (c *Context) String(code int, format string, args ...any) error {
	return c.Render(code, &render.String{Format: format, Data: args})
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

func (c *Context) Render(code int, render render.Render) error {
	c.W.WriteHeader(code)
	render.SetContentType(c.W)
	return render.Render(c.W)
}
