package raptor

import (
	"fmt"
	"net/http"
	"reflect"
	"time"
)

type coordinator struct {
	utils    *Utils
	handlers map[string]map[string]*handler
}

func newCoordinator(u *Utils) *coordinator {
	return &coordinator{
		utils:    u,
		handlers: make(map[string]map[string]*handler),
	}
}

func (c *coordinator) handle(ctx *Context) error {
	startTime := time.Now()
	c.logActionStart(ctx)
	err := c.handlers[ctx.Controller][ctx.Action].action(ctx)
	c.logActionFinish(ctx, startTime)
	return err
}

func (c *coordinator) logActionStart(ctx *Context) {
	c.utils.Log.Info(fmt.Sprintf("Started %s \"%s\" for %s", ctx.Request().Method, ctx.Request().URL.Path, ctx.RealIP()))
	c.utils.Log.Info(fmt.Sprintf("Processing by %s", actionDescriptor(ctx.Controller, ctx.Action)))
}

func (c *coordinator) logActionFinish(ctx *Context, startTime time.Time) {
	c.utils.Log.Info(fmt.Sprintf("Completed %d %s in %dms", ctx.Response().Status, http.StatusText(ctx.Response().Status), time.Since(startTime).Milliseconds()))
}

func (c *coordinator) registerController(controller interface{}, u *Utils, s map[string]ServiceInterface) error {
	controllerValue := reflect.ValueOf(controller)
	if err := c.validateController(controllerValue); err != nil {
		return err
	}

	controllerElem := controllerValue.Elem()
	controllerName := controllerElem.Type().Name()
	controllerElem.FieldByName("Controller").Addr().Interface().(*Controller).Init(u, s)

	if err := c.registerControllerActions(controllerValue, controllerName); err != nil {
		return err
	}

	return c.injectServices(controllerValue, controllerName, s)
}

func (c *coordinator) validateController(val reflect.Value) error {
	if val.Kind() != reflect.Pointer || val.Elem().FieldByName("Controller").Type() != reflect.TypeOf(Controller{}) {
		c.utils.Log.Error("Error while registering controller", "controller", val.Type().Name(), "error", "controller must be a pointer to a struct that embeds raptor.Controller")
		return fmt.Errorf("controller must be a pointer to a struct that embeds raptor.Controller")
	}
	return nil
}

func (c *coordinator) registerControllerActions(val reflect.Value, controller string) error {
	if c.handlers[controller] == nil {
		c.handlers[controller] = make(map[string]*handler)
	}

	for i := 0; i < val.NumMethod(); i++ {
		method := val.Method(i)
		methodType := method.Type()

		if c.isValidActionMethod(methodType) {
			action := val.Type().Method(i).Name
			c.handlers[controller][action] = newHandler(method.Interface().(func(*Context) error))
		}
	}

	return nil
}

func (c *coordinator) isValidActionMethod(methodType reflect.Type) bool {
	return methodType.NumIn() == 1 &&
		methodType.In(0) == reflect.TypeOf(&Context{}) &&
		methodType.NumOut() == 1 &&
		methodType.Out(0) == reflect.TypeOf((*error)(nil)).Elem()
}

func (c *coordinator) hasControllerAction(controller, action string) bool {
	_, ok := c.handlers[controller][action]
	return ok
}

func (c *coordinator) injectServices(controllerValue reflect.Value, controller string, services map[string]ServiceInterface) error {
	controllerElem := controllerValue.Elem()

	for i := 0; i < controllerElem.NumField(); i++ {
		field := controllerElem.Field(i)
		fieldType := controllerElem.Type().Field(i)

		if fieldType.Type.Kind() != reflect.Ptr ||
			fieldType.Type.Elem().Kind() != reflect.Struct {
			continue
		}

		if fieldType.Name == "Controller" {
			continue
		}

		service := fieldType.Type.Elem().Name()
		if injectedService, ok := services[service]; ok {
			field.Set(reflect.ValueOf(injectedService))
			continue
		}

		serviceInterfaceType := reflect.TypeOf((*ServiceInterface)(nil)).Elem()
		if fieldType.Type.Implements(serviceInterfaceType) {
			return fmt.Errorf("%s requires %s, but the service was not found in services initializer", controller, service)
		}
	}

	return nil
}

func (c *coordinator) injectMiddlewareGlobal(i int) error {
	for _, actions := range c.handlers {
		for _, handler := range actions {
			handler.injectMiddleware(uint8(i))
		}
	}

	return nil
}

func (c *coordinator) injectMiddlewareExcept(i int, exceptionDescriptors []string) error {
	excluded := make(map[string]struct{})
	for _, exception := range exceptionDescriptors {
		controller, action := parseActionDescriptor(exception)
		if !c.hasControllerAction(controller, action) {
			return fmt.Errorf("action %s#%s does not exist", controller, action)
		}
		excluded[actionDescriptor(controller, action)] = struct{}{}
	}

	for controller, actions := range c.handlers {
		for action, handler := range actions {
			if _, isExcluded := excluded[actionDescriptor(controller, action)]; !isExcluded {
				handler.injectMiddleware(uint8(i))
			}
		}
	}

	return nil
}

func (c *coordinator) injectMiddlewareOnly(i int, onlyDescriptors []string) error {
	included := make(map[string]struct{})
	for _, include := range onlyDescriptors {
		controller, action := parseActionDescriptor(include)
		if !c.hasControllerAction(controller, action) {
			return fmt.Errorf("action %s#%s does not exist", controller, action)
		}
		included[actionDescriptor(controller, action)] = struct{}{}
	}

	for controller, actions := range c.handlers {
		for action, handler := range actions {
			if _, isIncluded := included[actionDescriptor(controller, action)]; isIncluded {
				handler.injectMiddleware(uint8(i))
			}
		}
	}

	return nil
}
