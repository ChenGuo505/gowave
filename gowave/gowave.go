package gowave

import (
	"fmt"
	gwlog "github.com/ChenGuo505/gowave/log"
	"github.com/ChenGuo505/gowave/render"
	"html/template"
	"net/http"
	"sync"
)

type HandlerFunc func(ctx *Context)

type MiddlewareFunc func(next HandlerFunc) HandlerFunc

type routerGroup struct {
	prefix      string
	routes      map[string]map[string]HandlerFunc
	middlewares []MiddlewareFunc
	trie        *Trie
	logger      *gwlog.Logger
}

func (g *routerGroup) Use(middlewares ...MiddlewareFunc) {
	g.middlewares = append(g.middlewares, middlewares...)
}

func (g *routerGroup) Handle(h HandlerFunc, ctx *Context) {
	if g.middlewares != nil {
		for _, middleware := range g.middlewares {
			h = middleware(h)
		}
	}
	h(ctx)
}

func (g *routerGroup) register(path string, handler HandlerFunc, method string, middlewares ...MiddlewareFunc) {
	_, ok := g.routes[path]
	if !ok {
		g.routes[path] = make(map[string]HandlerFunc)
	}
	_, ok = g.routes[path][method]
	if ok {
		//log.Fatalf("duplicate handler for %s, method: %s", path, method)
		g.logger.Error(fmt.Sprintf("duplicate handler for %s, method: %s", path, method))
	}
	for _, middleware := range middlewares {
		handler = middleware(handler)
	}
	g.routes[path][method] = handler
	g.trie.Put(path)
}

func (g *routerGroup) Any(path string, handler HandlerFunc, middlewares ...MiddlewareFunc) {
	g.register(path, handler, http.MethodGet, middlewares...)
	g.register(path, handler, http.MethodPost, middlewares...)
	g.register(path, handler, http.MethodPut, middlewares...)
	g.register(path, handler, http.MethodDelete, middlewares...)
	g.register(path, handler, http.MethodOptions, middlewares...)
	g.register(path, handler, http.MethodPatch, middlewares...)
	g.register(path, handler, http.MethodHead, middlewares...)
}

func (g *routerGroup) Get(path string, handler HandlerFunc, middlewares ...MiddlewareFunc) {
	g.register(path, handler, http.MethodGet, middlewares...)
}

func (g *routerGroup) Post(path string, handler HandlerFunc, middlewares ...MiddlewareFunc) {
	g.register(path, handler, http.MethodPost, middlewares...)
}

func (g *routerGroup) Put(path string, handler HandlerFunc, middlewares ...MiddlewareFunc) {
	g.register(path, handler, http.MethodPut, middlewares...)
}

func (g *routerGroup) Delete(path string, handler HandlerFunc, middlewares ...MiddlewareFunc) {
	g.register(path, handler, http.MethodDelete, middlewares...)
}

func (g *routerGroup) Options(path string, handler HandlerFunc, middlewares ...MiddlewareFunc) {
	g.register(path, handler, http.MethodOptions, middlewares...)
}

func (g *routerGroup) Patch(path string, handler HandlerFunc, middlewares ...MiddlewareFunc) {
	g.register(path, handler, http.MethodPatch, middlewares...)
}

func (g *routerGroup) Head(path string, handler HandlerFunc, middlewares ...MiddlewareFunc) {
	g.register(path, handler, http.MethodHead, middlewares...)
}

type router struct {
	routerGroups []*routerGroup
	engine       *Engine
}

func (r *router) Group(prefix string) *routerGroup {
	routerGroup := &routerGroup{
		prefix: prefix,
		routes: make(map[string]map[string]HandlerFunc),
		trie:   NewTrie(),
		logger: r.engine.Logger,
	}
	routerGroup.Use(r.engine.middlewares...)
	r.routerGroups = append(r.routerGroups, routerGroup)
	return routerGroup
}

type Engine struct {
	router
	funcMap     template.FuncMap
	HTMLRender  render.HTMLRender
	pool        sync.Pool
	Logger      *gwlog.Logger
	middlewares []MiddlewareFunc
}

func New() *Engine {
	engine := &Engine{
		router: router{},
	}
	engine.pool.New = func() any {
		return engine.allocateContext()
	}
	engine.Logger = gwlog.DefaultLogger()
	engine.middlewares = []MiddlewareFunc{Logging, Recovery}
	engine.router.engine = engine
	return engine
}

func (e *Engine) allocateContext() *Context {
	return &Context{engine: e}
}

func (e *Engine) SetFuncMap(funcMap template.FuncMap) {
	e.funcMap = funcMap
}

func (e *Engine) SetHTMLRender(template *template.Template) {
	e.HTMLRender = render.HTMLRender{Template: template}
}

func (e *Engine) LoadTemplate(pattern string) {
	t := template.Must(template.New("").Funcs(e.funcMap).ParseGlob(pattern))
	e.SetHTMLRender(t)
}

func (e *Engine) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	ctx := e.pool.Get().(*Context)
	ctx.W = w
	ctx.Req = req
	ctx.Logger = e.Logger
	e.handleRequest(ctx, w, req)
	e.pool.Put(ctx)
}

func (e *Engine) handleRequest(ctx *Context, w http.ResponseWriter, req *http.Request) {
	e.Logger.Info(fmt.Sprintf("path: %s, method: %s", req.URL.Path, req.Method))
	for _, group := range e.routerGroups {
		routerName := TrimPrefix(req.URL.Path, "/"+group.prefix)
		node := group.trie.Get(routerName)
		if node != nil && node.isEnd {
			handler, ok := group.routes[node.routerName][req.Method]
			if ok {
				group.Handle(handler, ctx)
				return
			}
			w.WriteHeader(http.StatusMethodNotAllowed)
			_, err := fmt.Fprintf(w, "405 Method Not Allowed")
			if err != nil {
				return
			}
			return
		}
	}
	w.WriteHeader(http.StatusNotFound)
	_, err := fmt.Fprintf(w, "404 Not Found")
	if err != nil {
		return
	}
}

func (e *Engine) handler() http.Handler {
	return e
}

func (e *Engine) Run() {
	http.Handle("/", e)

	e.Logger.Info("Listening on :8080")
	err := http.ListenAndServe(":8080", nil)
	if err != nil {
		e.Logger.Fatal(fmt.Sprintf("Failed to start server: %v", err))
		return
	}
}

func (e *Engine) RunWithTLS(addr, certFile, keyFile string) {
	err := http.ListenAndServeTLS(addr, certFile, keyFile, e.handler())
	if err != nil {
		e.Logger.Fatal(fmt.Sprintf("Failed to start server with TLS: %v", err))
		return
	}
}
