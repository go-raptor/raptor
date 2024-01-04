package raptor

import (
	"reflect"
)

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
			if method.Type().NumIn() == 1 && method.Type().In(0) == reflect.TypeOf(&Context{}) && method.Type().NumOut() == 1 && method.Type().Out(0) == reflect.TypeOf((*error)(nil)).Elem() {
				controller.registerAction(val.Type().Method(i).Name, method.Interface().(func(*Context) error))
			}
		}

		return controller
	} else {
		panic("Controller must be a pointer to a struct that embeds raptor.Controller")
	}
}
