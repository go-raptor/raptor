package raptor

type Controllers []interface{}

type ControllerInterface interface {
	SetUtils(u *Utils)
}

type Controller struct {
	Utils *Utils
}

func (c *Controller) SetUtils(u *Utils) {
	c.Utils = u
}
