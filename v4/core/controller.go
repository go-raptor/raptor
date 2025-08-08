package core

import (
	"fmt"
	"reflect"

	"strings"

	"github.com/go-raptor/raptor/v4/errs"
)

const controllerSuffix = "Controller"
const descriptorSeparator = "."

type Controllers []ControllerInitializer

type ControllerInitializer interface {
	Init(*Resources)
}

type Controller struct {
	*Resources
	onInit func()
}

func (c *Controller) Init(resources *Resources) {
	c.Resources = resources
	if c.onInit != nil {
		c.onInit()
	}
}

func (c *Controller) OnInit(callback func()) {
	c.onInit = callback
}

func NormalizeController(controller string) string {
	if !strings.HasSuffix(controller, controllerSuffix) {
		return controller + controllerSuffix
	}
	return controller
}

func ParseActionDescriptor(descriptor string) (controller, action string) {
	parts := strings.Split(descriptor, descriptorSeparator)
	if len(parts) == 2 {
		return NormalizeController(parts[0]), parts[1]
	}
	return NormalizeController(descriptor), ""
}

func ActionDescriptor(controller, action string) string {
	return NormalizeController(controller) + descriptorSeparator + action
}

func NormalizeDescriptor(descriptor string) string {
	controller, action := ParseActionDescriptor(descriptor)
	if action == "" {
		return controller
	}
	return controller + descriptorSeparator + action
}

func (c *Core) RegisterControllers(components *Components) error {
	c.registerController(&ErrorsController{}, "ErrorsController")

	for _, controller := range components.Controllers {
		controllerName := reflect.TypeOf(controller).Elem().Name()
		if err := c.registerController(controller, controllerName); err != nil {
			c.Resources.Log.Error("Error while registering controller", "controller", controllerName, "error", err)
			return err
		}
	}

	return nil
}

func (c *Core) registerController(controller ControllerInitializer, controllerName string) error {
	if err := c.validateController(controller, controllerName); err != nil {
		return err
	}

	controller.Init(c.Resources)
	if err := c.registerControllerActions(reflect.ValueOf(controller), controllerName); err != nil {
		return err
	}

	if err := c.injectServices(controller, controllerName, "controller"); err != nil {
		return err
	}

	return nil
}

func (c *Core) validateController(controller interface{}, controllerName string) error {
	val := reflect.ValueOf(controller)
	if val.Kind() != reflect.Ptr || val.IsNil() {
		err := fmt.Errorf("controller must be a non-nil pointer to a struct")
		c.Resources.Log.Error("Error while registering controller", "controller", controllerName, "error", err)
		return err
	}
	if val.Elem().FieldByName("Controller").Type() != reflect.TypeOf(Controller{}) {
		err := fmt.Errorf("controller must embed raptor.Controller")
		c.Resources.Log.Error("Error while registering controller", "controller", controllerName, "error", err)
		return err
	}
	return nil
}

func (c *Core) registerControllerActions(val reflect.Value, controllerName string) error {
	for i := 0; i < val.NumMethod(); i++ {
		method := val.Method(i)
		methodType := method.Type()

		if c.isValidActionMethod(methodType) {
			action := val.Type().Method(i).Name
			c.RegisterHandler(controllerName, action, method.Interface().(func(*Context) error))
		}
	}
	return nil
}

func (c *Core) isValidActionMethod(methodType reflect.Type) bool {
	return methodType.NumIn() == 1 &&
		methodType.In(0) == reflect.TypeOf((*Context)(nil)) &&
		methodType.NumOut() == 1 &&
		methodType.Out(0) == reflect.TypeOf((*error)(nil)).Elem()
}

func (c *Core) RegisterHandler(controller, action string, handler HandlerFunc) {
	if c.Handlers[controller] == nil {
		c.Handlers[controller] = make(map[string]*Handler)
	}
	c.Handlers[controller][action] = &Handler{
		Action: handler,
	}
}

func (c *Core) HasControllerAction(controller, action string) bool {
	_, ok := c.Handlers[controller][action]
	return ok
}

type ErrorsController struct {
	Controller
}

func (e *ErrorsController) NotFound(ctx *Context) error {
	return errs.NewErrorNotFound(fmt.Sprintf("Handler not found for %s %s", ctx.Request().Method, ctx.Request().URL.Path))
}

func (e *ErrorsController) MethodNotAllowed(ctx *Context) error {
	ctx.response.Header().Set("Allow", ctx.Get("allowedMethods").(string))
	return errs.NewErrorMethodNotAllowed(fmt.Sprintf("Method %s not allowed for %s", ctx.Request().Method, ctx.Request().URL.Path))
}
