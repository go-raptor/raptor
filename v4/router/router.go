package router

import (
	"fmt"
	"net/http"
	"regexp"
	"slices"
	"strings"

	"github.com/go-raptor/raptor/v4/core"
)

var standardMethods = []string{
	"GET", "POST", "PUT", "DELETE", "PATCH", "HEAD", "OPTIONS", "CONNECT", "TRACE",
}

var pathRegex = regexp.MustCompile(`/+`)

type Router struct {
	Routes Routes
	Mux    *http.ServeMux
}

func NewRouter() *Router {
	return &Router{
		Mux: http.NewServeMux(),
	}
}

func (r *Router) RegisterRoutes(routes Routes, c *core.Core) error {
	r.Routes = routes
	for _, route := range r.Routes {
		if isHttpMethod(route.Method) {
			if _, exists := c.Handlers[route.Controller][route.Action]; !exists {
				return fmt.Errorf("action %s not found for %s %s", core.ActionDescriptor(route.Controller, route.Action), route.Method, route.Path)
			}
			route.core = c
			if err := r.registerRoute(&route); err != nil {
				return err
			}
		} else {
			return fmt.Errorf("invalid method %s on %s", route.Method, route.Path)
		}
	}
	return r.RegisterErrorHandlers(c)
}

func (r *Router) RegisterErrorHandlers(c *core.Core) error {
	pathMethods := make(map[string]map[string]struct{})
	hasCatchAllRoute := false

	for _, route := range r.Routes {
		if route.Pattern() == "/" {
			hasCatchAllRoute = true
		}
		if route.Method == "*" || route.Method == "ANY" {
			continue
		}
		if _, exists := pathMethods[route.Path]; !exists {
			pathMethods[route.Path] = make(map[string]struct{})
		}
		pathMethods[route.Path][route.Method] = struct{}{}
	}

	for path, allowed := range pathMethods {
		var allowedMethods []string
		for allowedMethod := range allowed {
			allowedMethods = append(allowedMethods, allowedMethod)
		}
		slices.Sort(allowedMethods)

		for _, method := range standardMethods {
			if _, exists := allowed[method]; !exists {
				store := map[string]interface{}{
					"allowedMethods": strings.Join(allowedMethods, ", "),
				}
				route := NewRoute(method, path, "ErrorsController", "MethodNotAllowed", store, c)
				if err := r.registerRoute(&route); err != nil {
					return err
				}
			}
		}
	}

	if !hasCatchAllRoute {
		route := NewRoute("ANY", "/", "ErrorsController", "NotFound", nil, c)
		if err := r.registerRoute(&route); err != nil {
			return err
		}
	}

	return nil
}

func (r *Router) registerRoute(route *Route) error {
	var err error
	defer func() {
		if r := recover(); r != nil {
			err = fmt.Errorf("error registering route %s: %v", route.Pattern(), r)
		}
	}()
	r.Mux.Handle(route.Pattern(), route)
	return err
}

func isHttpMethod(method string) bool {
	return slices.Contains(standardMethods, method) || method == "ANY" || method == "*"
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
		controller, action = core.ParseActionDescriptor(handler[0])
	} else if len(handler) == 2 {
		controller, action = handler[0], handler[1]
	}

	return Routes{
		NewRoute(method, normalizePath(path), controller, action, nil),
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
	return MethodRoute("ANY", path, handler...)
}

func CollectRoutes(r ...Routes) Routes {
	var result Routes
	for _, route := range r {
		result = append(result, route...)
	}
	return result
}
