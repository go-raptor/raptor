package raptor

import (
	"fmt"
	"net/http"
	"time"
)

type Controller struct {
	Name    string
	Utils   *Utils
	Actions map[string]action
}

func (c *Controller) SetUtils(r *Raptor) {
	c.Utils = r.Utils
}

func (c *Controller) Action(ctx *Context) error {
	startTime := time.Now()
	c.logActionStart(ctx)
	action := ctx.Locals("Action").(string)
	err := c.Actions[action].Function(ctx)
	c.logActionFinish(ctx, startTime)
	return err
}

func (c *Controller) logActionStart(ctx *Context) {
	action := ctx.Locals("Action").(string)
	c.Utils.Log.Info(fmt.Sprintf("Started %s \"%s\" for %s", ctx.Method(), ctx.OriginalURL(), ctx.IP()))
	c.Utils.Log.Info(fmt.Sprintf("Processing by %s#%s", c.Name, action))
}

func (c *Controller) logActionFinish(ctx *Context, startTime time.Time) {
	c.Utils.Log.Info(fmt.Sprintf("Completed %d %s in %dms", ctx.Response().StatusCode(), http.StatusText(ctx.Response().StatusCode()), time.Since(startTime).Milliseconds()))
}

func (c *Controller) registerAction(name string, function func(*Context) error) {
	if c.Actions == nil {
		c.Actions = make(map[string]action)
	}
	c.Actions[name] = action{
		Name:     name,
		Function: function,
	}
}
