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

func (r *Router) AddRoute(method string, path string, handler func(*Context) error) {
	/*route := Route{
		Method:  method,
		Path:    path,
		Handler: handler,
	}*/
	//r.Routes = append(r.Routes, route)
}
