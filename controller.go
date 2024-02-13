package raptor

type Controllers []interface{}

type ControllerInterface interface {
	Init(u *Utils)
	SetServices(s map[string]ServiceInterface)
}

type Controller struct {
	Utils    *Utils
	Services map[string]ServiceInterface
	onInit   func()
}

func (c *Controller) Init(u *Utils) {
	c.Utils = u
	if c.onInit != nil {
		c.onInit()
	}
}

func (c *Controller) SetServices(s map[string]ServiceInterface) {
	c.Services = s
}

func (c *Controller) OnInit(callback func()) {
	c.onInit = callback
}
