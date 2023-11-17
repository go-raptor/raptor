package raptor

type Controller struct {
	Services *Services
}

func (c *Controller) SetServices(r *Raptor) {
	c.Services = r.Services
}
