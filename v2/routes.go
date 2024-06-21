package raptor

type Routes []route

type route struct {
	Method     string
	Path       string
	Controller string
	Action     string
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

func CollectRoutes(r ...Routes) Routes {
	var result Routes
	for _, route := range r {
		result = append(result, route...)
	}
	return result
}
