package core

import (
	"fmt"
	"reflect"
	"time"

	"github.com/go-raptor/components"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
)

type echoMiddleware struct {
	middleware echo.MiddlewareFunc
	utils      *components.Utils
}

func (m *echoMiddleware) InitMiddleware(u *components.Utils) {
	m.utils = u
}

func (m *echoMiddleware) New(c components.State, next func(components.State) error) error {
	return m.middleware(func(ec echo.Context) error {
		return next(c)
	})(c.(echo.Context))
}

func Use(middleware components.MiddlewareProvider) components.ScopedMiddleware {
	return components.ScopedMiddleware{
		Middleware: middleware,
		Global:     true,
	}
}

func UseEcho(m echo.MiddlewareFunc) components.MiddlewareProvider {
	return &echoMiddleware{middleware: m}
}

func UseExcept(middleware components.MiddlewareProvider, except ...string) components.ScopedMiddleware {
	return components.ScopedMiddleware{
		Middleware: middleware,
		Except:     except,
	}
}

func UseOnly(middleware components.MiddlewareProvider, only ...string) components.ScopedMiddleware {
	return components.ScopedMiddleware{
		Middleware: middleware,
		Only:       only,
	}
}

func (c *Core) RegisterMiddlewares(server *echo.Echo, components *Components) error {
	c.registerCoreMiddlewares(server)

	for i, scopedMiddleware := range components.Middlewares {
		scopedMiddleware.Middleware.InitMiddleware(c.utils)
		c.middlewares = append(c.middlewares, scopedMiddleware.Middleware)
		var err error
		if scopedMiddleware.Global {
			err = c.injectMiddlewareGlobal(i)
		} else if scopedMiddleware.Except != nil {
			err = c.injectMiddlewareExcept(i, scopedMiddleware.Except)
		} else if scopedMiddleware.Only != nil {
			err = c.injectMiddlewareOnly(i, scopedMiddleware.Only)
		}
		if err != nil {
			c.utils.Log.Error("Error while registering middleware", "middleware", reflect.TypeOf(scopedMiddleware.Middleware).Elem().Name(), "error", err)
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

func (c *Core) registerCoreMiddlewares(server *echo.Echo) {
	server.Use(middleware.CORSWithConfig(middleware.CORSConfig{
		AllowOrigins:     c.utils.Config.CORSConfig.AllowOrigins,
		AllowHeaders:     c.utils.Config.CORSConfig.AllowHeaders,
		AllowMethods:     c.utils.Config.CORSConfig.AllowMethods,
		AllowCredentials: c.utils.Config.CORSConfig.AllowCredentials,
		MaxAge:           c.utils.Config.CORSConfig.MaxAge,
	}))

	if c.utils.Config.StaticConfig.Enabled {
		if c.utils.Config.StaticConfig.HTML5 {
			server.Use(middleware.StaticWithConfig(middleware.StaticConfig{
				Root:   c.utils.Config.StaticConfig.Root,
				Index:  c.utils.Config.StaticConfig.Index,
				Browse: c.utils.Config.StaticConfig.Browse,
				HTML5:  c.utils.Config.StaticConfig.HTML5,
			}))
		} else {
			server.Static(c.utils.Config.StaticConfig.Prefix, c.utils.Config.StaticConfig.Root)
		}
	}

	if c.utils.Config.GeneralConfig.LogRequests {
		server.Use(func(next echo.HandlerFunc) echo.HandlerFunc {
			return func(echoCtx echo.Context) error {
				startTime := time.Now()
				err := next(echoCtx)
				if err != nil {
					c.utils.Log.Error("Error while processing request", "error", err)
				}
				c.logRequest(GetContext(echoCtx), startTime)
				return err
			}
		})
	}

	server.Use(func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(echoCtx echo.Context) error {
			raptorCtx := GetContext(echoCtx)
			return next(raptorCtx)
		}
	})
}
