package core

import (
	"fmt"
	"reflect"

	"github.com/go-raptor/raptor/v4/components"
)

func (c *Core) RegisterControllers(components *components.Components) error {
	for _, controller := range components.Controllers {
		controllerName := reflect.TypeOf(controller).Elem().Name()
		if err := c.registerController(controller, controllerName); err != nil {
			c.Resources.Log.Error("Error while registering controller", "controller", controllerName, "error", err)
			return err
		}
	}
	return nil
}

func (c *Core) registerController(controller components.ControllerInitializer, controllerName string) error {
	if err := c.validateController(controller, controllerName); err != nil {
		return err
	}

	controller.Init(c.Resources)
	if err := c.registerControllerActions(reflect.ValueOf(controller), controllerName); err != nil {
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
	if val.Elem().FieldByName("Controller").Type() != reflect.TypeOf(components.Controller{}) {
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

func (c *Core) RegisterHandler(controller, action string, handler func(*Context) error) {
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
