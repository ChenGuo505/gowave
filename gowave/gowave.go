package gowave

import (
	"fmt"
	"log"
	"net/http"
)

type HandlerFunc func(ctx *Context)

type MiddlewareFunc func(next HandlerFunc) HandlerFunc

type routerGroup struct {
	prefix      string
	routes      map[string]map[string]HandlerFunc
	middlewares []MiddlewareFunc
	trie        *Trie
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
		log.Fatalf("duplicate handler for %s, method: %s", path, method)
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
}

func (r *router) Group(prefix string) *routerGroup {
	routerGroup := &routerGroup{
		prefix: prefix,
		routes: make(map[string]map[string]HandlerFunc),
		trie:   NewTrie(),
	}
	r.routerGroups = append(r.routerGroups, routerGroup)
	return routerGroup
}

type Engine struct {
	router
}

func New() *Engine {
	return &Engine{
		router: router{},
	}
}

func (e *Engine) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	e.handleRequest(w, req)
}

func (e *Engine) handleRequest(w http.ResponseWriter, req *http.Request) {
	log.Printf("path: %s, method: %s", req.URL.Path, req.Method)
	for _, group := range e.routerGroups {
		routerName := TrimPrefix(req.RequestURI, "/"+group.prefix)
		node := group.trie.Get(routerName)
		if node != nil && node.isEnd {
			handler, ok := group.routes[node.routerName][req.Method]
			if ok {
				ctx := &Context{
					W:   w,
					Req: req,
				}
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

func (e *Engine) Run() {
	http.Handle("/", e)

	log.Println("Starting server on :8080")
	err := http.ListenAndServe(":8080", nil)
	if err != nil {
		log.Fatal(err)
	}
}
