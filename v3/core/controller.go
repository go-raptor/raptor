package core

import (
	"fmt"
	"reflect"
	"strings"

	"github.com/go-raptor/raptor/v3/components"
)

const controllerSuffix = "Controller"
const descriptorSeparator = "."

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
	return controller + descriptorSeparator + action
}

func (c *Core) RegisterControllers(components *Components) error {
	for _, controller := range components.Controllers {
		if err := c.registerController(controller); err != nil {
			c.utils.Log.Error("Error while registering controller", "controller", reflect.TypeOf(controller).Elem().Name(), "error", err)
			return err
		}
	}

	return nil
}

func (c *Core) registerController(controller interface{}) error {
	controllerValue := reflect.ValueOf(controller)
	if err := c.validateController(controllerValue); err != nil {
		return err
	}

	controllerElem := controllerValue.Elem()
	controllerName := controllerElem.Type().Name()
	controllerElem.FieldByName("Controller").Addr().Interface().(*components.Controller).Init(c.utils)

	if err := c.registerControllerActions(controllerValue, controllerName); err != nil {
		return err
	}

	return c.injectServicesToController(controllerValue, controllerName)
}

func (c *Core) validateController(val reflect.Value) error {
	if val.Kind() != reflect.Pointer || val.Elem().FieldByName("Controller").Type() != reflect.TypeOf(components.Controller{}) {
		c.utils.Log.Error("Error while registering controller", "controller", val.Type().Name(), "error", "controller must be a pointer to a struct that embeds raptor.Controller")
		return fmt.Errorf("controller must be a pointer to a struct that embeds raptor.Controller")
	}
	return nil
}

func (c *Core) registerControllerActions(val reflect.Value, controller string) error {
	if c.handlers[controller] == nil {
		c.handlers[controller] = make(map[string]*handler)
	}

	for i := 0; i < val.NumMethod(); i++ {
		method := val.Method(i)
		methodType := method.Type()

		if c.isValidActionMethod(methodType) {
			action := val.Type().Method(i).Name
			c.handlers[controller][action] = newHandler(method.Interface().(func(*components.Context) error))
		}
	}

	return nil
}

func (c *Core) isValidActionMethod(methodType reflect.Type) bool {
	return methodType.NumIn() == 1 &&
		methodType.In(0) == reflect.TypeOf(&components.Context{}) &&
		methodType.NumOut() == 1 &&
		methodType.Out(0) == reflect.TypeOf((*error)(nil)).Elem()
}

func (c *Core) HasControllerAction(controller, action string) bool {
	_, ok := c.handlers[controller][action]
	return ok
}
