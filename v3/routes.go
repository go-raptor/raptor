package raptor

import (
	"regexp"
	"strings"
)

const controllerSuffix = "Controller"

var pathRegex = regexp.MustCompile(`/+`)

type Routes []route

type route struct {
	Method     string
	Path       string
	Controller string
	Action     string
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

func normalizeController(controller string) string {
	if !strings.HasSuffix(controller, controllerSuffix) {
		return controller + controllerSuffix
	}
	return controller
}

func parseControllerAction(input string) (controller, action string) {
	parts := strings.Split(input, "#")
	if len(parts) == 2 {
		return normalizeController(parts[0]), parts[1]
	}
	return normalizeController(input), ""
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

func Route(method, path string, handler ...string) Routes {
	var controller, action string

	if len(handler) == 1 {
		controller, action = parseControllerAction(handler[0])
	} else if len(handler) == 2 {
		controller, action = handler[0], handler[1]
	}

	return Routes{
		route{
			Method:     method,
			Path:       normalizePath(path),
			Controller: normalizeController(controller),
			Action:     action,
		},
	}
}

func Get(path string, handler ...string) Routes {
	return Route("GET", path, handler...)
}

func Post(path string, handler ...string) Routes {
	return Route("POST", path, handler...)
}

func Put(path string, handler ...string) Routes {
	return Route("PUT", path, handler...)
}

func Patch(path string, handler ...string) Routes {
	return Route("PATCH", path, handler...)
}

func Delete(path string, handler ...string) Routes {
	return Route("DELETE", path, handler...)
}

func Options(path string, handler ...string) Routes {
	return Route("OPTIONS", path, handler...)
}

func Head(path string, handler ...string) Routes {
	return Route("HEAD", path, handler...)
}

func Connect(path string, handler ...string) Routes {
	return Route("CONNECT", path, handler...)
}

func Trace(path string, handler ...string) Routes {
	return Route("TRACE", path, handler...)
}

func Propfind(path string, handler ...string) Routes {
	return Route("PROPFIND", path, handler...)
}

func Report(path string, handler ...string) Routes {
	return Route("REPORT", path, handler...)
}

func Any(path string, handler ...string) Routes {
	return Route("*", path, handler...)
}

func CollectRoutes(r ...Routes) Routes {
	var result Routes
	for _, route := range r {
		result = append(result, route...)
	}
	return result
}
