package router

import (
	"fmt"
	"net/http"
	"regexp"

	"github.com/go-raptor/errs"
	"github.com/go-raptor/raptor/v3/core"
	"github.com/labstack/echo/v4"
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
}

func New(routes Routes, c *core.Core, server *echo.Echo) (*Router, error) {
	router := &Router{
		Routes: routes,
	}

	for _, route := range router.Routes {
		if !c.HasControllerAction(route.Controller, route.Action) {
			return nil, fmt.Errorf("action %s not found in controller %s for path %s", route.Action, route.Controller, route.Path)
		}

		handler := func(echoCtx echo.Context) error {
			echoCtx.Set(string(core.ControllerKey), route.Controller)
			echoCtx.Set(string(core.ActionKey), route.Action)
			return c.Handle(echoCtx)
		}

		if route.Method != "*" {
			server.Add(route.Method, route.Path, handler)
		} else {
			server.Any(route.Path, handler)
		}
	}

	hasCatchAll := false
	for _, route := range router.Routes {
		if route.Path == "/*" {
			hasCatchAll = true
			break
		}
	}
	if !hasCatchAll {
		server.Any("/*", func(ctx echo.Context) error {
			return errs.NewError(http.StatusNotFound, "Handler not found",
				"method", ctx.Request().Method,
				"path", ctx.Request().URL.Path,
			)
		})
	}

	return router, nil
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
	var controller, action string

	if len(actionDescriptor) == 1 {
		controller, action = core.ParseActionDescriptor(actionDescriptor[0])
	} else if len(actionDescriptor) == 2 {
		controller, action = actionDescriptor[0], actionDescriptor[1]
	}

	return Routes{
		Route{
			Method:     method,
			Path:       normalizePath(path),
			Controller: core.NormalizeController(controller),
			Action:     action,
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
