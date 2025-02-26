package components

type Controllers []interface{}

type ControllerInterface interface {
	Init(u *Utils)
}

type Controller struct {
	*Utils
	onInit func()
}

func (c *Controller) Init(u *Utils) {
	c.Utils = u
	if c.onInit != nil {
		c.onInit()
	}
}

func (c *Controller) OnInit(callback func()) {
	c.onInit = callback
}
