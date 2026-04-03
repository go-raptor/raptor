package core

import (
	"fmt"
	"net/http"
	"reflect"
)

var middlewareType = reflect.TypeFor[Middleware]()

type ScopedMiddleware struct {
	Middleware MiddlewareInitializer
	Only       []string
	Except     []string
	Global     bool
}

type Middlewares []ScopedMiddleware

type MiddlewareInitializer interface {
	Init(*Resources)
	Handle(c *Context, next func(*Context) error) error
}

// MiddlewareSetup is an optional interface that middlewares can implement
// to perform initialization after resources have been injected.
type MiddlewareSetup interface {
	Setup() error
}

type Middleware struct {
	*Resources
}

func (m *Middleware) Init(resources *Resources) {
	m.Resources = resources
}

func Use(middleware MiddlewareInitializer) ScopedMiddleware {
	return ScopedMiddleware{
		Middleware: middleware,
		Global:     true,
	}
}

func UseExcept(middleware MiddlewareInitializer, except ...string) ScopedMiddleware {
	return ScopedMiddleware{
		Middleware: middleware,
		Except:     except,
	}
}

func UseOnly(middleware MiddlewareInitializer, only ...string) ScopedMiddleware {
	return ScopedMiddleware{
		Middleware: middleware,
		Only:       only,
	}
}

func (c *Core) RegisterMiddlewares(components *Components) error {
	c.Middlewares = make([]MiddlewareInitializer, 0, len(components.Middlewares))

	for _, scoped := range components.Middlewares {
		middlewareName := reflect.TypeOf(scoped.Middleware).Elem().Name()

		if err := c.registerMiddleware(scoped, middlewareName); err != nil {
			return err
		}
	}

	return nil
}

func (c *Core) registerMiddleware(scoped ScopedMiddleware, middlewareName string) error {
	if err := c.validateMiddleware(scoped.Middleware, middlewareName); err != nil {
		return err
	}

	if err := c.validateScope(scoped, middlewareName); err != nil {
		return err
	}

	scoped.Middleware.Init(c.Resources)

	if setup, ok := scoped.Middleware.(MiddlewareSetup); ok {
		if err := setup.Setup(); err != nil {
			c.Resources.Log.Error("Middleware setup failed", "middleware", middlewareName, "error", err)
			return err
		}
	}

	c.Middlewares = append(c.Middlewares, scoped.Middleware)

	if err := c.injectServices(scoped.Middleware, middlewareName, "middleware"); err != nil {
		return err
	}

	c.applyMiddleware(len(c.Middlewares)-1, scoped)
	return nil
}

func (c *Core) validateMiddleware(middleware any, middlewareName string) error {
	val := reflect.ValueOf(middleware)
	if val.Kind() != reflect.Pointer || val.IsNil() {
		return fmt.Errorf("%s: middleware must be a non-nil pointer to a struct", middlewareName)
	}

	field := val.Elem().FieldByName("Middleware")
	if !field.IsValid() || field.Type() != middlewareType {
		return fmt.Errorf("%s: middleware must embed raptor.Middleware", middlewareName)
	}

	return nil
}

func (c *Core) validateScope(scoped ScopedMiddleware, middlewareName string) error {
	hasGlobal := scoped.Global
	hasOnly := len(scoped.Only) > 0
	hasExcept := len(scoped.Except) > 0

	if hasGlobal == hasOnly || hasGlobal == hasExcept || hasOnly == hasExcept {
		if hasGlobal && hasOnly {
			return fmt.Errorf("%s: middleware scoping must specify exactly one of Global, Only, or Except", middlewareName)
		}
		if !hasGlobal && !hasOnly && !hasExcept {
			return fmt.Errorf("%s: middleware must specify one of Global, Only, or Except", middlewareName)
		}
	}

	descriptors := scoped.Only
	scopeType := "Only"
	if hasExcept {
		descriptors = scoped.Except
		scopeType = "Except"
	}

	for _, descriptor := range descriptors {
		controller, action := ParseActionDescriptor(descriptor)

		if _, ok := c.Handlers[controller]; !ok {
			return fmt.Errorf("%s: controller '%s' in %s scope does not exist", middlewareName, controller, scopeType)
		}

		if action != "" && !c.HasControllerAction(controller, action) {
			return fmt.Errorf("%s: action '%s.%s' in %s scope does not exist", middlewareName, controller, action, scopeType)
		}
	}

	return nil
}

func (c *Core) applyMiddleware(middlewareIndex int, scoped ScopedMiddleware) {
	for controller, actions := range c.Handlers {
		for action, handler := range actions {
			if matchesScope(scoped, controller, action) {
				handler.injectMiddleware(middlewareIndex)
			}
		}
	}
}

func matchesScope(scoped ScopedMiddleware, handlerController, handlerAction string) bool {
	if scoped.Global {
		return true
	}

	if len(scoped.Only) > 0 {
		return matchesDescriptors(scoped.Only, handlerController, handlerAction)
	}

	return !matchesDescriptors(scoped.Except, handlerController, handlerAction)
}

func matchesDescriptors(descriptors []string, handlerController, handlerAction string) bool {
	for _, descriptor := range descriptors {
		ruleController, ruleAction := ParseActionDescriptor(descriptor)
		if ruleController != handlerController {
			continue
		}
		if ruleAction == "" || ruleAction == handlerAction {
			return true
		}
	}
	return false
}

type stdMiddlewareAdapter struct {
	Middleware
	wrap func(http.Handler) http.Handler
}

func (a *stdMiddlewareAdapter) Handle(ctx *Context, next func(*Context) error) error {
	var nextErr error
	nextHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx.SetRequest(r)
		nextErr = next(ctx)
	})
	a.wrap(nextHandler).ServeHTTP(ctx.Response(), ctx.Request())
	return nextErr
}

func UseStd(mw func(http.Handler) http.Handler) ScopedMiddleware {
	return Use(&stdMiddlewareAdapter{wrap: mw})
}

func UseStdOnly(mw func(http.Handler) http.Handler, only ...string) ScopedMiddleware {
	return UseOnly(&stdMiddlewareAdapter{wrap: mw}, only...)
}

func UseStdExcept(mw func(http.Handler) http.Handler, except ...string) ScopedMiddleware {
	return UseExcept(&stdMiddlewareAdapter{wrap: mw}, except...)
}
