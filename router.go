package raptor

type Routes []Route

func RegisterRoutes(routes ...Route) []Route {
	return routes
}

type Route struct {
	Method     string
	Path       string
	Controller string
	Action     string
}

func NewRoute(method, path, controller, action string) Route {
	return Route{
		Method:     method,
		Path:       path,
		Controller: controller,
		Action:     action,
	}
}
