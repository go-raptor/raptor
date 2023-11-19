package raptor

type Router struct {
	ControllerRoutes []ControllerRoute
}

type ControllerRoute struct {
	Name       string
	Controller Controller
	Routes     []Route
}

type Route struct {
	Method  string
	Path    string
	Handler func(*Context) error
}

func NewRouter() *Router {
	return &Router{
		ControllerRoutes: make([]ControllerRoute, 0),
	}
}

func (r *Router) AddControllerRoute(name string, controller Controller, routes ...Route) {
	r.ControllerRoutes = append(r.ControllerRoutes, ControllerRoute{
		Name:       name,
		Controller: controller,
		Routes:     routes,
	})
}

func NewRoute(method string, path string, handler func(*Context) error) Route {
	return Route{
		Method:  method,
		Path:    path,
		Handler: handler,
	}
}
