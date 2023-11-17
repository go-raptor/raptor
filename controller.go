package raptor

type Controller interface {
	SetServices(r *Raptor)
}

type DefaultController struct {
	Services *Services
}

func (c *DefaultController) SetServices(r *Raptor) {
	c.Services = r.Services
}
