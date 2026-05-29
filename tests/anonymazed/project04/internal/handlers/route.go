package handlers

import (
	"net/http"
)

type Route struct {
	Path   string
	Handler http.HandlerFunc
	Meth   []string
}

type Router struct {
	routes []Route
}

func NewRouter() *Router {
	return &Router{
		routes: make([]Route, 0),
	}
}

func (r *Router) Add(path string, handler http.HandlerFunc, methods []string) {
	r.routes = append(r.routes, Route{
		Path:   path,
		Handler: handler,
		Meth:   methods,
	})
}

func (r *Router) Match(path string, method string) (http.HandlerFunc, bool) {
	for _, route := range r.routes {
		for _, m := range route.Meth {
			if m == method && route.Path == path {
				return route.Handler, true
			}
		}
	}
	return nil, false
}

func (r *Router) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	handler, found := r.Match(req.URL.Path, req.Method)
	if !found {
		http.NotFound(w, req)
		return
	}
	handler(w, req)
}
