package raptor

import "errors"

type Routes []route

type route struct {
	Method     string
	Path       string
	Controller string
	Action     string
}

func Scope(path string, routes ...Routes) Routes {
	var result Routes
	for _, route := range routes {
		for _, r := range route {
			r.Path = path + r.Path
			result = append(result, r)
		}
	}
	return result
}

func Route(method, path, controller, action string) Routes {
	return Routes{
		route{
			Method:     method,
			Path:       path,
			Controller: controller,
			Action:     action,
		},
	}
}

func Get(path, controller, action string) Routes {
	return Route("GET", path, controller, action)
}

func Post(path, controller, action string) Routes {
	return Route("POST", path, controller, action)
}

func Put(path, controller, action string) Routes {
	return Route("PUT", path, controller, action)
}

func Patch(path, controller, action string) Routes {
	return Route("PATCH", path, controller, action)
}

func Delete(path, controller, action string) Routes {
	return Route("DELETE", path, controller, action)
}

func Options(path, controller, action string) Routes {
	return Route("OPTIONS", path, controller, action)
}

func Head(path, controller, action string) Routes {
	return Route("HEAD", path, controller, action)
}

func Connect(path, controller, action string) Routes {
	return Route("CONNECT", path, controller, action)
}

func Trace(path, controller, action string) Routes {
	return Route("TRACE", path, controller, action)
}

func Propfind(path, controller, action string) Routes {
	return Route("PROPFIND", path, controller, action)
}

func Report(path, controller, action string) Routes {
	return Route("REPORT", path, controller, action)
}

func Any(path, controller, action string) Routes {
	return Route("*", path, controller, action)
}

func CollectRoutes(r ...Routes) Routes {
	var result Routes
	for _, route := range r {
		result = append(result, route...)
	}
	return result
}

func (r *Routes) Path(controller, action string) (string, error) {
	for _, route := range *r {
		if route.Controller == controller && route.Action == action {
			return route.Path, nil
		}
	}

	return "", errors.New("route not found")
}
