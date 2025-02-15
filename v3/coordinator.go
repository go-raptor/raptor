package raptor

import (
	"fmt"
	"net/http"
	"reflect"
	"sync"
	"time"

	"github.com/labstack/echo/v4"
)

type coordinator struct {
	utils       *Utils
	handlers    map[string]map[string]*handler
	contextPool sync.Pool
	services    map[string]ServiceInterface
	middlewares []MiddlewareInterface
	routes      Routes
}

func newCoordinator(u *Utils) *coordinator {
	return &coordinator{
		utils:    u,
		handlers: make(map[string]map[string]*handler),
		contextPool: sync.Pool{
			New: func() interface{} {
				return new(Context)
			},
		},
		services:    make(map[string]ServiceInterface),
		middlewares: make([]MiddlewareInterface, 0),
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

func (c *coordinator) registerControllers(app *AppInitializer) error {
	for _, controller := range app.Controllers {
		if err := c.registerController(controller); err != nil {
			c.utils.Log.Error("Error while registering controller", "controller", reflect.TypeOf(controller).Elem().Name(), "error", err)
			return err
		}
	}

	return nil
}

func (c *coordinator) registerController(controller interface{}) error {
	controllerValue := reflect.ValueOf(controller)
	if err := c.validateController(controllerValue); err != nil {
		return err
	}

	controllerElem := controllerValue.Elem()
	controllerName := controllerElem.Type().Name()
	controllerElem.FieldByName("Controller").Addr().Interface().(*Controller).Init(c.utils)

	if err := c.registerControllerActions(controllerValue, controllerName); err != nil {
		return err
	}

	return c.injectServicesToController(controllerValue, controllerName)
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

func (c *coordinator) injectServicesToController(controllerValue reflect.Value, controller string) error {
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
		if injectedService, ok := c.services[service]; ok {
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

func (c *coordinator) registerServices(app *AppInitializer) error {
	for _, service := range app.Services {
		if err := service.InitService(c.utils); err != nil {
			c.utils.Log.Error("Service initialization failed", "service", reflect.TypeOf(service).Elem().Name(), "error", err)
			return err
		}
		c.services[reflect.TypeOf(service).Elem().Name()] = service
	}

	for _, service := range c.services {
		if err := c.injectServicesToService(service); err != nil {
			return err
		}
	}

	return nil
}

func (c *coordinator) injectServicesToService(service ServiceInterface) error {
	serviceValue := reflect.ValueOf(service).Elem()
	serviceType := reflect.TypeOf(service).Elem()

	for i := 0; i < serviceValue.NumField(); i++ {
		field := serviceValue.Field(i)
		fieldType := serviceType.Field(i)

		if fieldType.Type.Kind() != reflect.Ptr || fieldType.Type.Elem().Kind() != reflect.Struct {
			continue
		}

		if injectedService, ok := c.services[fieldType.Type.Elem().Name()]; ok {
			field.Set(reflect.ValueOf(injectedService))
			continue
		}

		serviceInterfaceType := reflect.TypeOf((*ServiceInterface)(nil)).Elem()
		if fieldType.Type.Implements(serviceInterfaceType) {
			err := fmt.Errorf("%s requires %s, but the service was not found in services initializer", serviceType.Name(), fieldType.Type.Elem().Name())
			c.utils.Log.Error("Error while registering service", "service", serviceType.Name(), "error", err)
			return err
		}
	}

	return nil
}

func (c *coordinator) registerMiddlewares(app *AppInitializer) error {
	for i, scopedMiddleware := range app.Middlewares {
		scopedMiddleware.middleware.InitMiddleware(c.utils)
		c.middlewares = append(c.middlewares, scopedMiddleware.middleware)
		var err error
		if scopedMiddleware.global {
			err = c.injectMiddlewareGlobal(i)
		} else if scopedMiddleware.except != nil {
			err = c.injectMiddlewareExcept(i, scopedMiddleware.except)
		} else if scopedMiddleware.only != nil {
			err = c.injectMiddlewareOnly(i, scopedMiddleware.only)
		}
		if err != nil {
			c.utils.Log.Error("Error while registering middleware", "middleware", reflect.TypeOf(scopedMiddleware.middleware).Elem().Name(), "error", err)
			return err
		}
	}

	for _, middleware := range c.middlewares {
		if err := c.injectServicesToMiddleware(middleware); err != nil {
			return err
		}
	}

	return nil
}

func (c *coordinator) injectServicesToMiddleware(middleware MiddlewareInterface) error {
	middlewareValue := reflect.ValueOf(middleware).Elem()
	middlewareType := reflect.TypeOf(middleware).Elem()

	for i := 0; i < middlewareValue.NumField(); i++ {
		field := middlewareValue.Field(i)
		fieldType := middlewareType.Field(i)

		if fieldType.Type.Kind() != reflect.Ptr || fieldType.Type.Elem().Kind() != reflect.Struct {
			continue
		}

		serviceName := fieldType.Type.Elem().Name()
		if injectedService, ok := c.services[serviceName]; ok {
			field.Set(reflect.ValueOf(injectedService))
			continue
		}

		serviceInterfaceType := reflect.TypeOf((*ServiceInterface)(nil)).Elem()
		if fieldType.Type.Implements(serviceInterfaceType) {
			err := fmt.Errorf("%s requires %s, but the service was not found in services initializer", middlewareType.Name(), serviceName)
			c.utils.Log.Error("Error while registering middleware", "middleware", middlewareType.Name(), "error", err)
			return err
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

func (c *coordinator) acquireContext(ec echo.Context, controller, action string) *Context {
	ctx := c.contextPool.Get().(*Context)
	ctx.Context = ec
	ctx.Controller = controller
	ctx.Action = action
	return ctx
}

func (c *coordinator) releaseContext(ctx *Context) {
	if ctx == nil {
		return
	}
	ctx.Context = nil
	ctx.Controller = ""
	ctx.Action = ""
	c.contextPool.Put(ctx)
}

func (c *coordinator) registerRoutes(app *AppInitializer, server *echo.Echo) error {
	c.routes = app.Routes
	for _, route := range c.routes {
		if !c.hasControllerAction(route.Controller, route.Action) {
			err := fmt.Errorf("action %s not found in controller %s for path %s", route.Action, route.Controller, route.Path)
			c.utils.Log.Error("Error while registering route", "error", err)
			return err
		}
		c.registerRoute(route, server)
	}

	return nil
}

func (c *coordinator) registerRoute(route route, server *echo.Echo) {
	routeHandler := c.CreateActionWrapper(route.Controller, route.Action, c.handle)
	if route.Method != "*" {
		server.Add(route.Method, route.Path, routeHandler)
		return
	}
	server.Any(route.Path, routeHandler)
}
