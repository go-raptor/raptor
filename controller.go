package raptor

type Controller struct {
	Name     string
	Services *Services
	Actions  map[string]action
}

func (c *Controller) SetServices(r *Raptor) {
	c.Services = r.Services
}

func NewController(name string, c *Controller, actions ...action) *Controller {
	c.Name = name
	c.registerActions(actions...)
	return c
}

func (c *Controller) registerActions(actions ...action) {
	if c.Actions == nil {
		c.Actions = make(map[string]action)
	}
	for _, action := range actions {
		c.Actions[action.Name] = action
	}
}

type Controllers map[string]*Controller

func NewControllers(controller ...*Controller) Controllers {
	controllers := make(Controllers)
	for _, c := range controller {
		controllers[c.Name] = c
	}
	return controllers
}
