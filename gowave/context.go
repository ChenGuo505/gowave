package gowave

import (
	"encoding/json"
	"errors"
	"github.com/ChenGuo505/gowave/render"
	"html/template"
	"io"
	"log"
	"mime/multipart"
	"net/http"
	"net/url"
	"os"
	"strings"
)

const (
	defaultMaxMemory = 32 << 20 // 32 MB
)

type Context struct {
	W   http.ResponseWriter
	Req *http.Request

	engine     *Engine
	queryCache url.Values
	formCache  url.Values

	DisallowUnknownFields bool // Disallow unknown fields in JSON parsing
}

func (c *Context) initQueryCache() {
	if c.Req != nil {
		c.queryCache = c.Req.URL.Query()
	} else {
		c.queryCache = make(url.Values)
	}
}

func (c *Context) GetQuery(key string) string {
	c.initQueryCache()
	return c.queryCache.Get(key)
}

func (c *Context) GetQueryAll(key string) []string {
	c.initQueryCache()
	values, _ := c.queryCache[key]
	return values
}

func (c *Context) GetQueryDefault(key, defaultValue string) string {
	values := c.GetQueryAll(key)
	if len(values) == 0 {
		return defaultValue
	}
	return values[0]
}

func (c *Context) GetQueryMap(key string) map[string]string {
	c.initQueryCache()
	return c.getFromMap(c.queryCache, key)
}

func (c *Context) initFormCache() {
	if c.Req != nil {
		if err := c.Req.ParseMultipartForm(defaultMaxMemory); err != nil {
			if errors.Is(err, http.ErrNotMultipart) {
				log.Fatal(err)
			}
		}
		c.formCache = c.Req.PostForm
	} else {
		c.formCache = make(url.Values)
	}
}

func (c *Context) GetForm(key string) string {
	c.initFormCache()
	return c.formCache.Get(key)
}

func (c *Context) GetFormAll(key string) []string {
	c.initFormCache()
	values, _ := c.formCache[key]
	return values
}

func (c *Context) GetFormDefault(key, defaultValue string) string {
	values := c.GetFormAll(key)
	if len(values) == 0 {
		return defaultValue
	}
	return values[0]
}

func (c *Context) GetFormMap(key string) map[string]string {
	c.initFormCache()
	return c.getFromMap(c.formCache, key)
}

func (c *Context) FormFile(name string) *multipart.FileHeader {
	file, header, err := c.Req.FormFile(name)
	if err != nil {
		log.Fatal(err)
	}
	defer func(file multipart.File) {
		err := file.Close()
		if err != nil {
			log.Fatal(err)
		}
	}(file)
	return header
}

func (c *Context) FormFiles(name string) []*multipart.FileHeader {
	multipartForm, err := c.MultipartForm()
	if err != nil {
		log.Println(err)
		return make([]*multipart.FileHeader, 0)
	}
	return multipartForm.File[name]
}

func (c *Context) SaveFile(file *multipart.FileHeader, dst string) error {
	src, err := file.Open()
	if err != nil {
		return err
	}
	defer func(src multipart.File) {
		err := src.Close()
		if err != nil {
			log.Fatal(err)
		}
	}(src)

	out, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer func(out *os.File) {
		err := out.Close()
		if err != nil {
			log.Fatal(err)
		}
	}(out)

	_, err = io.Copy(out, src)
	return err
}

func (c *Context) MultipartForm() (*multipart.Form, error) {
	err := c.Req.ParseMultipartForm(defaultMaxMemory)
	return c.Req.MultipartForm, err
}

func (c *Context) getFromMap(cache map[string][]string, key string) map[string]string {
	dict := make(map[string]string)
	for k, v := range cache {
		if i := strings.Index(k, "["); i >= 1 && k[0:i] == key {
			if j := strings.Index(k[i+1:], "]"); j >= 1 {
				dict[k[i+1:][:j]] = v[0]
			}
		}
	}
	return dict
}

func (c *Context) ParseJson(obj any) error {
	body := c.Req.Body
	if body == nil {
		return errors.New("request body is nil")
	}
	decoder := json.NewDecoder(body)
	if c.DisallowUnknownFields {
		decoder.DisallowUnknownFields()
	}
	return decoder.Decode(obj)
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
	render.SetContentType(c.W)
	err := render.Render(c.W)
	if code != http.StatusOK {
		c.W.WriteHeader(code)
	}
	return err
}
