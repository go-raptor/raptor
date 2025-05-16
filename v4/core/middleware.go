package core

import (
	"fmt"
	"reflect"
)

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

type Middleware struct {
	*Resources
	onInit func()
}

func (m *Middleware) Init(resources *Resources) {
	m.Resources = resources
	if m.onInit != nil {
		m.onInit()
	}
}

func (m *Middleware) OnInit(callback func()) {
	m.onInit = callback
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
	var errs []error
	c.Middlewares = make([]MiddlewareInitializer, 0, len(components.Middlewares))

	for _, scopedMiddleware := range components.Middlewares {
		middlewareName := reflect.TypeOf(scopedMiddleware.Middleware).Elem().Name()

		if err := c.validateMiddleware(scopedMiddleware.Middleware, middlewareName); err != nil {
			c.Resources.Log.Error("Error while registering middleware", "middleware", middlewareName, "error", err)
			errs = append(errs, err)
			continue
		}

		if err := c.validateScopedMiddleware(scopedMiddleware, middlewareName); err != nil {
			c.Resources.Log.Error("Error while registering middleware", "middleware", middlewareName, "error", err)
			errs = append(errs, err)
			continue
		}

		if err := c.registerMiddleware(scopedMiddleware.Middleware); err != nil {
			c.Resources.Log.Error("Error while registering middleware", "middleware", middlewareName, "error", err)
			errs = append(errs, err)
			continue
		}

		middlewareIndex := len(c.Middlewares) - 1
		if err := c.injectServices(scopedMiddleware.Middleware, middlewareName, "middleware"); err != nil {
			c.Resources.Log.Error("Error while injecting services into middleware", "middleware", middlewareName, "error", err)
			errs = append(errs, err)
			continue
		}

		if err := c.injectMiddleware(middlewareIndex, scopedMiddleware); err != nil {
			c.Resources.Log.Error("Error while injecting middleware", "middleware", middlewareName, "error", err)
			errs = append(errs, err)
			continue
		}
	}

	if len(errs) > 0 {
		return fmt.Errorf("multiple errors registering middlewares: %v", errs)
	}
	return nil
}

func (c *Core) validateMiddleware(middleware interface{}, middlewareName string) error {
	val := reflect.ValueOf(middleware)
	if val.Kind() != reflect.Ptr || val.IsNil() {
		return fmt.Errorf("%s: middleware must be a non-nil pointer to a struct", middlewareName)
	}
	if !val.Type().Implements(reflect.TypeOf((*MiddlewareInitializer)(nil)).Elem()) {
		return fmt.Errorf("%s: middleware must implement MiddlewareInitializer", middlewareName)
	}
	if val.Elem().FieldByName("Middleware").Type() != reflect.TypeOf(Middleware{}) {
		return fmt.Errorf("%s: middleware must embed core.Middleware", middlewareName)
	}
	return nil
}

func (c *Core) validateScopedMiddleware(scoped ScopedMiddleware, middlewareName string) error {
	hasGlobal := scoped.Global
	hasOnly := len(scoped.Only) > 0
	hasExcept := len(scoped.Except) > 0

	count := 0
	if hasGlobal {
		count++
	}
	if hasOnly {
		count++
	}
	if hasExcept {
		count++
	}
	if count > 1 {
		return fmt.Errorf("%s: middleware scoping must specify exactly one of Global, Only, or Except", middlewareName)
	}
	if count == 0 {
		return fmt.Errorf("%s: middleware must specify one of Global, Only, or Except", middlewareName)
	}

	if hasOnly {
		for _, descriptor := range scoped.Only {
			controller, action := ParseActionDescriptor(descriptor)
			if !c.HasControllerAction(controller, action) {
				return fmt.Errorf("%s: action %s#%s in Only does not exist", middlewareName, controller, action)
			}
		}
	}
	if hasExcept {
		for _, descriptor := range scoped.Except {
			controller, action := ParseActionDescriptor(descriptor)
			if !c.HasControllerAction(controller, action) {
				return fmt.Errorf("%s: action %s#%s in Except does not exist", middlewareName, controller, action)
			}
		}
	}
	return nil
}

func (c *Core) registerMiddleware(middleware MiddlewareInitializer) error {
	middleware.Init(c.Resources)
	c.Middlewares = append(c.Middlewares, middleware)
	return nil
}

// injectMiddleware applies a middleware to handlers based on scoping rules.
func (c *Core) injectMiddleware(middlewareIndex int, scoped ScopedMiddleware) error {
	shouldInclude := func(controller, action string) bool {
		descriptor := ActionDescriptor(controller, action)
		if scoped.Global {
			return true
		}
		if len(scoped.Only) > 0 {
			for _, only := range scoped.Only {
				if NormalizeDescriptor(only) == descriptor {
					return true
				}
			}
			return false
		}
		if len(scoped.Except) > 0 {
			for _, except := range scoped.Except {
				if NormalizeDescriptor(except) == descriptor {
					return false
				}
			}
			return true
		}
		return false
	}

	for controller, actions := range c.Handlers {
		for action, handler := range actions {
			if shouldInclude(controller, action) {
				handler.injectMiddleware(middlewareIndex)
			}
		}
	}
	return nil
}
