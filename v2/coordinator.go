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
	c.utils.Log.Info(fmt.Sprintf("Started %s \"%s\" for %s", ctx.Method(), ctx.OriginalURL(), ctx.IP()))
	c.utils.Log.Info(fmt.Sprintf("Processing by %s#%s", ctx.Controller, ctx.Action))
}

func (c *coordinator) logActionFinish(ctx *Context, startTime time.Time) {
	c.utils.Log.Info(fmt.Sprintf("Completed %d %s in %dms", ctx.Response().StatusCode(), http.StatusText(ctx.Response().StatusCode()), time.Since(startTime).Milliseconds()))
}

func (c *coordinator) registerController(controller interface{}, u *Utils, s map[string]ServiceInterface) {
	val := reflect.ValueOf(controller)

	if val.Kind() == reflect.Pointer && val.Elem().FieldByName("Controller").Type() == reflect.TypeOf(Controller{}) {
		val.Elem().FieldByName("Controller").Addr().Interface().(*Controller).Init(u, s)
		controllerName := val.Elem().Type().Name()
		if c.actions[controllerName] == nil {
			c.actions[controllerName] = make(map[string]func(*Context) error)
		}

		for i := 0; i < val.NumMethod(); i++ {
			method := val.Method(i)
			if method.Type().NumIn() == 1 && method.Type().In(0) == reflect.TypeOf(&Context{}) && method.Type().NumOut() == 1 && method.Type().Out(0) == reflect.TypeOf((*error)(nil)).Elem() {
				c.actions[controllerName][val.Type().Method(i).Name] = method.Interface().(func(*Context) error)
			}
		}

		for i := 0; i < reflect.ValueOf(controller).Elem().NumField(); i++ {
			field := reflect.ValueOf(controller).Elem().Field(i)
			fieldType := reflect.TypeOf(controller).Elem().Field(i)
			if fieldType.Type.Kind() == reflect.Ptr && fieldType.Type.Elem().Kind() == reflect.Struct {
				if service, ok := s[fieldType.Type.Elem().Name()]; ok {
					field.Set(reflect.ValueOf(service))
				}
			}
		}
	} else {
		panic("Controller must be a pointer to a struct that embeds raptor.Controller")
	}
}
