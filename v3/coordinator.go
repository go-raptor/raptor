package raptor

import (
	"fmt"
	"net/http"
	"reflect"
	"time"
)

type coordinator struct {
	utils   *Utils
	actions map[string]map[string]func(*Context) error
}

func newCoordinator(u *Utils) *coordinator {
	return &coordinator{
		utils:   u,
		actions: make(map[string]map[string]func(*Context) error),
	}
}

func (c *coordinator) action(ctx *Context) error {
	startTime := time.Now()
	c.logActionStart(ctx)
	err := c.actions[ctx.Controller][ctx.Action](ctx)
	c.logActionFinish(ctx, startTime)
	return err
}

func (c *coordinator) logActionStart(ctx *Context) {
	c.utils.Log.Info(fmt.Sprintf("Started %s \"%s\" for %s", ctx.Request().Method, ctx.Request().URL.Path, ctx.RealIP()))
	c.utils.Log.Info(fmt.Sprintf("Processing by %s#%s", ctx.Controller, ctx.Action))
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

	if err := c.registerActions(controllerValue, controllerName); err != nil {
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

func (c *coordinator) registerActions(val reflect.Value, controllerName string) error {
	if c.actions[controllerName] == nil {
		c.actions[controllerName] = make(map[string]func(*Context) error)
	}

	for i := 0; i < val.NumMethod(); i++ {
		method := val.Method(i)
		methodType := method.Type()

		if isValidActionMethod(methodType) {
			actionName := val.Type().Method(i).Name
			c.actions[controllerName][actionName] = method.Interface().(func(*Context) error)
		}
	}

	return nil
}

func isValidActionMethod(methodType reflect.Type) bool {
	return methodType.NumIn() == 1 &&
		methodType.In(0) == reflect.TypeOf(&Context{}) &&
		methodType.NumOut() == 1 &&
		methodType.Out(0) == reflect.TypeOf((*error)(nil)).Elem()
}

func (c *coordinator) injectServices(controllerValue reflect.Value, controllerName string, services map[string]ServiceInterface) error {
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

		serviceName := fieldType.Type.Elem().Name()
		if injectedService, ok := services[serviceName]; ok {
			field.Set(reflect.ValueOf(injectedService))
			continue
		}

		serviceInterfaceType := reflect.TypeOf((*ServiceInterface)(nil)).Elem()
		if fieldType.Type.Implements(serviceInterfaceType) {
			return fmt.Errorf("%s requires %s, but the service was not found in services initializer", controllerName, serviceName)
		}
	}

	return nil
}
