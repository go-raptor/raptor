package router

import (
	"net/http"

	"github.com/go-raptor/raptor/v4/core"
)

type Routes []Route

type Route struct {
	core       *core.Core
	Store      map[string]any
	Method     string
	Path       string
	Controller string
	Action     string
}

func NewRoute(method, path, controller, action string, store map[string]any) Route {
	return Route{
		Store:      store,
		Method:     method,
		Path:       path,
		Controller: controller,
		Action:     action,
	}
}

func (r *Route) Pattern() string {
	if r.Method == "ANY" {
		return r.Path
	}
	return r.Method + " " + r.Path
}

func (r *Route) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	r.core.Handler(w, req, r.Controller, r.Action, r.Path, r.Store)
}
