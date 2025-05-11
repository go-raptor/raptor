package router

import (
	"net/http"

	"github.com/go-raptor/raptor/v4/core"
)

type Routes []Route

type Route struct {
	core       *core.Core
	Method     string
	Path       string
	Controller string
	Action     string
}

func NewRoute(method, path, controller, action string, c ...*core.Core) Route {
	var core *core.Core
	if len(c) > 0 {
		core = c[0]
	}

	return Route{
		core:       core,
		Method:     method,
		Path:       path,
		Controller: controller,
		Action:     action,
	}
}

func (route *Route) Pattern() string {
	return route.Method + " " + route.Path
}

func (route *Route) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	route.core.Handler(w, r, route.Controller, route.Action, route.Path)
}
