package raptor

type Routes []route

type route struct {
	Method     string
	Path       string
	Controller string
	Action     string
}

func Route(method, path, controller, action string) route {
	return route{
		Method:     method,
		Path:       path,
		Controller: controller,
		Action:     action,
	}
}
