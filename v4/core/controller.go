package core

import (
	"fmt"
	"reflect"
	"strings"

	"github.com/go-raptor/raptor/v4/errs"
)

const controllerSuffix = "Controller"
const descriptorSeparator = "."

var (
	contextPtrType = reflect.TypeFor[*Context]()
	errorType      = reflect.TypeFor[error]()
	controllerType = reflect.TypeFor[Controller]()
)

type Controllers []ControllerInitializer

type ControllerInitializer interface {
	Init(*Resources)
}

// ControllerSetup is an optional interface that controllers can implement
// to perform initialization after resources have been injected.
type ControllerSetup interface {
	Setup() error
}

type Controller struct {
	*Resources
}

func (c *Controller) Init(resources *Resources) {
	c.Resources = resources
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

	if setup, ok := controller.(ControllerSetup); ok {
		if err := setup.Setup(); err != nil {
			c.Resources.Log.Error("Controller setup failed", "controller", controllerName, "error", err)
			return err
		}
	}

	c.registerControllerActions(reflect.ValueOf(controller), controllerName)

	return c.injectServices(controller, controllerName, "controller")
}

func (c *Core) validateController(controller any, controllerName string) error {
	val := reflect.ValueOf(controller)
	if val.Kind() != reflect.Pointer || val.IsNil() {
		err := fmt.Errorf("controller must be a non-nil pointer to a struct")
		c.Resources.Log.Error("Error while registering controller", "controller", controllerName, "error", err)
		return err
	}

	field := val.Elem().FieldByName("Controller")
	if !field.IsValid() || field.Type() != controllerType {
		err := fmt.Errorf("controller must embed raptor.Controller")
		c.Resources.Log.Error("Error while registering controller", "controller", controllerName, "error", err)
		return err
	}

	return nil
}

func (c *Core) registerControllerActions(val reflect.Value, controllerName string) {
	for i := 0; i < val.NumMethod(); i++ {
		method := val.Method(i)
		if isActionMethod(method.Type()) {
			action := val.Type().Method(i).Name
			c.RegisterHandler(controllerName, action, method.Interface().(func(*Context) error))
		}
	}
}

func isActionMethod(t reflect.Type) bool {
	return t.NumIn() == 1 &&
		t.In(0) == contextPtrType &&
		t.NumOut() == 1 &&
		t.Out(0) == errorType
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
	return errs.NewErrorMethodNotAllowed(fmt.Sprintf("Method %s not allowed for %s", ctx.Request().Method, ctx.Request().URL.Path))
}
