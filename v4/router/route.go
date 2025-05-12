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

func (r *Route) Pattern() string {
	return r.Method + " " + r.Path
}

func (r *Route) ServeHTTP(writer http.ResponseWriter, request *http.Request) {
	r.core.Handler(writer, request, r.Controller, r.Action, r.Path)
}
