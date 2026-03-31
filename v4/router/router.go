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
	for i := range r.Routes {
		route := &r.Routes[i]
		if !isHTTPMethod(route.Method) {
			return fmt.Errorf("invalid method %s on %s", route.Method, route.Path)
		}
		if _, exists := c.Handlers[route.Controller][route.Action]; !exists {
			return fmt.Errorf("action %s not found for %s %s", core.ActionDescriptor(route.Controller, route.Action), route.Method, route.Path)
		}
		route.core = c
		r.Mux.Handle(route.Pattern(), route)
	}
	return r.registerErrorHandlers(c)
}

func (r *Router) registerErrorHandlers(c *core.Core) error {
	pathMethods := make(map[string]map[string]struct{})
	hasCatchAll := false

	for _, route := range r.Routes {
		if route.Pattern() == "/" {
			hasCatchAll = true
		}
		if route.Method == "*" || route.Method == "ANY" {
			continue
		}
		if pathMethods[route.Path] == nil {
			pathMethods[route.Path] = make(map[string]struct{})
		}
		pathMethods[route.Path][route.Method] = struct{}{}
	}

	for _, methods := range pathMethods {
		if _, hasGet := methods["GET"]; hasGet {
			methods["HEAD"] = struct{}{}
		}
	}

	for path, allowed := range pathMethods {
		allowedList := slices.Sorted(func(yield func(string) bool) {
			for method := range allowed {
				if !yield(method) {
					return
				}
			}
		})
		allowedStr := strings.Join(allowedList, ", ")

		for _, method := range standardMethods {
			if _, exists := allowed[method]; exists {
				continue
			}
			route := NewRoute(method, path, "ErrorsController", "MethodNotAllowed", map[string]any{
				"allowedMethods": allowedStr,
			})
			route.core = c
			r.Mux.Handle(route.Pattern(), &route)
		}
	}

	if !hasCatchAll {
		route := NewRoute("ANY", "/", "ErrorsController", "NotFound", nil)
		route.core = c
		r.Mux.Handle(route.Pattern(), &route)
	}

	return nil
}

func isHTTPMethod(method string) bool {
	return slices.Contains(standardMethods, method) || method == "ANY" || method == "*"
}

func Scope(path string, routes ...Routes) Routes {
	parentPath := normalizePath(path)
	var result Routes
	for _, routeSet := range routes {
		for _, r := range routeSet {
			r.Path = normalizePath(parentPath + "/" + r.Path)
			result = append(result, r)
		}
	}
	return result
}

func MethodRoute(method, path, descriptor string) Routes {
	controller, action := core.ParseActionDescriptor(descriptor)
	return Routes{
		NewRoute(method, normalizePath(path), controller, action, nil),
	}
}

func normalizePath(path string) string {
	if path == "" || path == "/" {
		return "/"
	}

	path = pathRegex.ReplaceAllString("/"+path, "/")
	if len(path) > 1 && strings.HasSuffix(path, "/") {
		path = path[:len(path)-1]
	}

	return path
}

func Get(path, handler string) Routes    { return MethodRoute("GET", path, handler) }
func Post(path, handler string) Routes   { return MethodRoute("POST", path, handler) }
func Put(path, handler string) Routes    { return MethodRoute("PUT", path, handler) }
func Patch(path, handler string) Routes  { return MethodRoute("PATCH", path, handler) }
func Delete(path, handler string) Routes { return MethodRoute("DELETE", path, handler) }

func Options(path, handler string) Routes { return MethodRoute("OPTIONS", path, handler) }
func Head(path, handler string) Routes    { return MethodRoute("HEAD", path, handler) }
func Connect(path, handler string) Routes { return MethodRoute("CONNECT", path, handler) }
func Trace(path, handler string) Routes   { return MethodRoute("TRACE", path, handler) }
func Any(path, handler string) Routes     { return MethodRoute("ANY", path, handler) }

func CollectRoutes(r ...Routes) Routes {
	var result Routes
	for _, route := range r {
		result = append(result, route...)
	}
	return result
}
