package raptor

import (
	"fmt"
	"net/http"
	"time"
)

type Controller struct {
	Name     string
	Services *Services
	Actions  map[string]action
}

func (c *Controller) SetServices(r *Raptor) {
	c.Services = r.Services
}

func (c *Controller) Action(ctx *Context) error {
	startTime := time.Now()
	c.logStart(ctx)
	action := ctx.Locals("Action").(string)
	err := c.Actions[action].Function(ctx)
	c.logFinish(ctx, startTime)
	return err
}

func (c *Controller) logStart(ctx *Context) {
	action := ctx.Locals("Action").(string)
	c.Services.Log.Info(fmt.Sprintf("Started %s \"%s\" for %s", ctx.Method(), ctx.OriginalURL(), ctx.IP()))
	c.Services.Log.Info(fmt.Sprintf("Processing by %sController#%s", c.Name, action))
}

func (c *Controller) logFinish(ctx *Context, startTime time.Time) {
	c.Services.Log.Info(fmt.Sprintf("Completed %d %s in %dms", ctx.Response().StatusCode(), http.StatusText(ctx.Response().StatusCode()), time.Since(startTime).Milliseconds()))
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

func RegisterController(name string, c *Controller, actions ...action) *Controller {
	c.Name = name
	c.registerActions(actions...)
	return c
}

func RegisterControllers(controller ...*Controller) Controllers {
	controllers := make(Controllers)
	for _, c := range controller {
		controllers[c.Name] = c
	}
	return controllers
}
