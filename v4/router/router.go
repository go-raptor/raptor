package router

import (
	"fmt"
	"net/http"
	"regexp"

	"github.com/go-raptor/raptor/v4/components"
	"github.com/go-raptor/raptor/v4/core"
)

var pathRegex = regexp.MustCompile(`/+`)

type Routes []Route

type Route struct {
	Method     string
	Path       string
	Controller string
	Action     string
	Handler    func(*core.Context) error
}

type Router struct {
	Routes Routes
	Mux    *http.ServeMux
}

func NewRouter() (*Router, error) {
	router := &Router{
		Mux: http.NewServeMux(),
	}
	return router, nil
}

func (r *Router) RegisterRoutes(routes Routes, c *core.Core) error {
	r.Routes = routes
	for _, route := range r.Routes {
		if isHttpMethod(route.Method) {
			routeHandler := &routeHandler{
				core:       c,
				controller: route.Controller,
				action:     route.Action,
			}
			r.Mux.Handle(route.Method+" "+route.Path, routeHandler)
		} else {
			return fmt.Errorf("invalid method %s on %s", route.Method, route.Path)
		}
	}
	return nil
}

func isHttpMethod(method string) bool {
	switch method {
	case "GET", "POST", "PUT", "DELETE", "PATCH", "HEAD", "OPTIONS", "CONNECT", "TRACE", "ANY", "*":
		return true
	default:
		return false
	}
}

func Scope(path string, routes ...Routes) Routes {
	var result Routes
	normalizedPath := normalizePath(path)

	for _, route := range routes {
		for _, r := range route {
			if r.Path == "/" {
				r.Path = normalizedPath
			} else {
				r.Path = normalizedPath + r.Path
			}
			result = append(result, r)
		}
	}
	return result
}

func MethodRoute(method, path string, handler ...string) Routes {
	var controller, action string

	if len(handler) == 1 {
		controller, action = components.ParseActionDescriptor(handler[0])
	} else if len(handler) == 2 {
		controller, action = handler[0], handler[1]
	}

	return Routes{
		Route{
			Method:     method,
			Path:       normalizePath(path),
			Controller: controller,
			Action:     action,
		},
	}
}

func normalizePath(path string) string {
	if path == "" {
		return "/"
	}

	path = pathRegex.ReplaceAllString("/"+path+"/", "/")
	if len(path) > 1 {
		path = path[:len(path)-1]
	}

	return path
}

func Get(path string, handler ...string) Routes {
	return MethodRoute("GET", path, handler...)
}

func Post(path string, handler ...string) Routes {
	return MethodRoute("POST", path, handler...)
}

func Put(path string, handler ...string) Routes {
	return MethodRoute("PUT", path, handler...)
}

func Patch(path string, handler ...string) Routes {
	return MethodRoute("PATCH", path, handler...)
}

func Delete(path string, handler ...string) Routes {
	return MethodRoute("DELETE", path, handler...)
}

func Options(path string, handler ...string) Routes {
	return MethodRoute("OPTIONS", path, handler...)
}

func Head(path string, handler ...string) Routes {
	return MethodRoute("HEAD", path, handler...)
}

func Connect(path string, handler ...string) Routes {
	return MethodRoute("CONNECT", path, handler...)
}

func Trace(path string, handler ...string) Routes {
	return MethodRoute("TRACE", path, handler...)
}

func Propfind(path string, handler ...string) Routes {
	return MethodRoute("PROPFIND", path, handler...)
}

func Report(path string, handler ...string) Routes {
	return MethodRoute("REPORT", path, handler...)
}

func Any(path string, handler ...string) Routes {
	return MethodRoute("*", path, handler...)
}

func CollectRoutes(r ...Routes) Routes {
	var result Routes
	for _, route := range r {
		result = append(result, route...)
	}
	return result
}
