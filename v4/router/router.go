package router

import (
	"net/http"
	"regexp"

	"github.com/go-raptor/raptor/v4/core"
)

var pathRegex = regexp.MustCompile(`/+`)

type Routes []Route

type Route struct {
	Method     string
	Path       string
	Controller string
	Action     string
}

type Router struct {
	Routes Routes
	Mux    *http.ServeMux
}

func New() (*Router, error) {
	router := &Router{
		Mux: http.NewServeMux(),
	}
	return router, nil
}

func (r *Router) RegisterRoutes(routes Routes, c *core.Core) error {
	r.Routes = routes
	for _, route := range r.Routes {
		c.Utils.Log.Info("Registering route", "method", route.Method, "path", route.Path, "controller", route.Controller, "action", route.Action)
		r.Mux.Handle(route.Method+" "+route.Path, c)
	}

	return nil
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

func MethodRoute(method, path string, actionDescriptor ...string) Routes {
	return Routes{
		Route{
			Method:     method,
			Path:       normalizePath(path),
			Controller: "",
			Action:     "",
		},
	}
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
