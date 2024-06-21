package raptor

type Controllers []interface{}

type ControllerInterface interface {
	Init(u *Utils, s map[string]ServiceInterface)
}

type Controller struct {
	*Utils
	Services map[string]ServiceInterface
	onInit   func()
}

func (c *Controller) Init(u *Utils, s map[string]ServiceInterface) {
	c.Utils = u
	c.Services = s
	if c.onInit != nil {
		c.onInit()
	}
}

func (c *Controller) OnInit(callback func()) {
	c.onInit = callback
}
