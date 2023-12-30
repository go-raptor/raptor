package raptor

import (
	"fmt"
	"net/http"
	"reflect"
	"runtime"
	"strings"
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
	c.logStart(ctx)
	action := ctx.Locals("Action").(string)
	err := c.Actions[action].Function(ctx)
	c.logFinish(ctx, startTime)
	return err
}

func (c *Controller) logStart(ctx *Context) {
	action := ctx.Locals("Action").(string)
	c.Utils.Log.Info(fmt.Sprintf("Started %s \"%s\" for %s", ctx.Method(), ctx.OriginalURL(), ctx.IP()))
	c.Utils.Log.Info(fmt.Sprintf("Processing by %sController#%s", c.Name, action))
}

func (c *Controller) logFinish(ctx *Context, startTime time.Time) {
	c.Utils.Log.Info(fmt.Sprintf("Completed %d %s in %dms", ctx.Response().StatusCode(), http.StatusText(ctx.Response().StatusCode()), time.Since(startTime).Milliseconds()))
}

func (c *Controller) registerActions(functions ...func(*Context) error) {
	if c.Actions == nil {
		c.Actions = make(map[string]action)
	}
	for _, function := range functions {
		fullName := runtime.FuncForPC(reflect.ValueOf(function).Pointer()).Name()
		parts := strings.Split(fullName, ".")
		name := parts[len(parts)-1]

		if strings.HasSuffix(name, "-fm") {
			name = strings.TrimSuffix(name, "-fm")
		}

		c.Actions[name] = action{
			Name:     name,
			Function: function,
		}
	}
}

type Controllers map[string]*Controller

func RegisterController(c interface{}, functions ...func(*Context) error) *Controller {
	val := reflect.ValueOf(c)

	if val.Kind() == reflect.Ptr && val.Elem().FieldByName("Controller").Type() == reflect.TypeOf(Controller{}) {
		name := val.Elem().Type().Name()
		if strings.HasSuffix(name, "Controller") {
			name = strings.TrimSuffix(name, "Controller")
		}
		controller := val.Elem().FieldByName("Controller").Addr().Interface().(*Controller)
		controller.Name = name
		controller.registerActions(functions...)
		return controller
	} else {
		panic("Controller must be a pointer to a struct that embeds raptor.Controller")
	}
}

func RegisterControllers(controller ...*Controller) Controllers {
	controllers := make(Controllers)
	for _, c := range controller {
		controllers[c.Name] = c
	}
	return controllers
}
