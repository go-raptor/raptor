package router

import (
	"regexp"
	"strings"
)

const controllerSuffix = "Controller"
const descriptorSeparator = "."

var pathRegex = regexp.MustCompile(`/+`)

type Routes []Route

type Route struct {
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

func ParseActionDescriptor(descriptor string) (controller, action string) {
	parts := strings.Split(descriptor, descriptorSeparator)
	if len(parts) == 2 {
		return normalizeController(parts[0]), parts[1]
	}
	return normalizeController(descriptor), ""
}

func ActionDescriptor(controller, action string) string {
	return controller + descriptorSeparator + action
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
		controller, action = ParseActionDescriptor(actionDescriptor[0])
	} else if len(actionDescriptor) == 2 {
		controller, action = actionDescriptor[0], actionDescriptor[1]
	}

	return Routes{
		Route{
			Method:     method,
			Path:       normalizePath(path),
			Controller: normalizeController(controller),
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
