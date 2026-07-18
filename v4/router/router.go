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
		h, exists := c.Handlers[route.Controller][route.Action]
		if !exists {
			return fmt.Errorf("action %s not found for %s %s", core.ActionDescriptor(route.Controller, route.Action), route.Method, route.Path)
		}
		route.core = c
		route.handler = h
		r.Mux.Handle(route.Pattern(), route)
	}
	return r.registerErrorHandlers(c)
}

// registerErrorHandlers installs a catch-all fallback that renders 404s and,
// when the path is served under other methods, 405s with an Allow header.
// Skipped when the app registered its own catch-all route on "/".
func (r *Router) registerErrorHandlers(c *core.Core) error {
	for _, route := range r.Routes {
		if route.Pattern() == "/" {
			return nil
		}
	}

	r.Mux.Handle("/", &fallbackHandler{
		mux:        r.Mux,
		core:       c,
		notFound:   c.Handlers["ErrorsController"]["NotFound"],
		notAllowed: c.Handlers["ErrorsController"]["MethodNotAllowed"],
	})
	return nil
}

// fallbackHandler serves every request no registered pattern matched. It
// probes the mux with each other standard method for the same path to
// distinguish 405 (another method matches) from 404 (none does).
type fallbackHandler struct {
	mux        *http.ServeMux
	core       *core.Core
	notFound   *core.Handler
	notAllowed *core.Handler
}

func (f *fallbackHandler) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	if allow := f.allowedMethods(req); allow != "" {
		w.Header().Set("Allow", allow)
		f.core.Serve(w, req, f.notAllowed, "ErrorsController", "MethodNotAllowed", "/", nil)
		return
	}
	f.core.Serve(w, req, f.notFound, "ErrorsController", "NotFound", "/", nil)
}

func (f *fallbackHandler) allowedMethods(req *http.Request) string {
	var allowed []string
	probe := new(http.Request)
	*probe = *req
	for _, method := range standardMethods {
		if method == req.Method {
			continue
		}
		probe.Method = method
		if _, pattern := f.mux.Handler(probe); pattern != "" && pattern != "/" {
			allowed = append(allowed, method)
		}
	}
	slices.Sort(allowed)
	return strings.Join(allowed, ", ")
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
