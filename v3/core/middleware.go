package core

import (
	"fmt"
	"reflect"

	"github.com/labstack/echo/v4"
)

func (m *Middleware) InitMiddleware(u *Utils) {
	m.Utils = u
	if m.onInit != nil {
		m.onInit()
	}
}

func (m *Middleware) OnInit(callback func()) {
	m.onInit = callback
}

func (m *echoMiddleware) New(c *Context) error {
	return m.handler(c.Context)
}

func (m *echoMiddleware) InitMiddleware(u *Utils) {
}

func UseEcho(h echo.HandlerFunc) *echoMiddleware {
	return &echoMiddleware{handler: h}
}

func Use(middleware MiddlewareInterface) ScopedMiddleware {
	return ScopedMiddleware{
		middleware: middleware,
		global:     true,
	}
}

func UseExcept(middleware MiddlewareInterface, except ...string) ScopedMiddleware {
	return ScopedMiddleware{
		middleware: middleware,
		except:     except,
	}
}

func UseOnly(middleware MiddlewareInterface, only ...string) ScopedMiddleware {
	return ScopedMiddleware{
		middleware: middleware,
		only:       only,
	}
}

func (c *Core) RegisterMiddlewares(components *Components) error {
	for i, scopedMiddleware := range components.Middlewares {
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

func (c *Core) injectMiddlewareGlobal(i int) error {
	for _, actions := range c.handlers {
		for _, handler := range actions {
			handler.injectMiddleware(uint8(i))
		}
	}

	return nil
}

func (c *Core) injectMiddlewareExcept(i int, exceptionDescriptors []string) error {
	excluded := make(map[string]struct{})
	for _, exception := range exceptionDescriptors {
		controller, action := ParseActionDescriptor(exception)
		if !c.HasControllerAction(controller, action) {
			return fmt.Errorf("action %s#%s does not exist", controller, action)
		}
		excluded[ActionDescriptor(controller, action)] = struct{}{}
	}

	for controller, actions := range c.handlers {
		for action, handler := range actions {
			if _, isExcluded := excluded[ActionDescriptor(controller, action)]; !isExcluded {
				handler.injectMiddleware(uint8(i))
			}
		}
	}

	return nil
}

func (c *Core) injectMiddlewareOnly(i int, onlyDescriptors []string) error {
	included := make(map[string]struct{})
	for _, include := range onlyDescriptors {
		controller, action := ParseActionDescriptor(include)
		if !c.HasControllerAction(controller, action) {
			return fmt.Errorf("action %s#%s does not exist", controller, action)
		}
		included[ActionDescriptor(controller, action)] = struct{}{}
	}

	for controller, actions := range c.handlers {
		for action, handler := range actions {
			if _, isIncluded := included[ActionDescriptor(controller, action)]; isIncluded {
				handler.injectMiddleware(uint8(i))
			}
		}
	}

	return nil
}
