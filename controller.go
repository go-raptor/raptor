package raptor

import (
	"fmt"
	"net/http"
	"reflect"
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

type Controllers map[string]*Controller

func RegisterControllers(controllers ...interface{}) Controllers {
	c := make(Controllers)
	for _, controller := range controllers {
		registeredController := registerController(controller)
		c[registeredController.Name] = registeredController
	}
	return c
}

func registerController(c interface{}) *Controller {
	val := reflect.ValueOf(c)

	if val.Kind() == reflect.Ptr && val.Elem().FieldByName("Controller").Type() == reflect.TypeOf(Controller{}) {
		controller := val.Elem().FieldByName("Controller").Addr().Interface().(*Controller)
		controller.Name = val.Elem().Type().Name()

		for i := 0; i < val.NumMethod(); i++ {
			method := val.Method(i)
			if method.Type().NumIn() == 1 && method.Type().In(0) == reflect.TypeOf(&Context{}) {
				methodName := val.Type().Method(i).Name
				if methodName != "Action" {
					controller.registerAction(methodName, method.Interface().(func(*Context) error))
				}
			}
		}
		return controller
	} else {
		panic("Controller must be a pointer to a struct that embeds raptor.Controller")
	}
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
