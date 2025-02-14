package raptor

import (
	"github.com/labstack/echo/v4"
)

func (c *coordinator) CreateActionWrapper(controller, action string, handler func(*Context) error) echo.HandlerFunc {
	return func(echoCtx echo.Context) error {
		ctx := c.acquireContext(echoCtx, controller, action)
		defer c.releaseContext(ctx)

		for _, middlewareIndex := range c.handlers[controller][action].middlewares {
			if err := c.middlewares[middlewareIndex].New(ctx); err != nil {
				return ctx.JSONError(err)
			}
		}

		return handler(ctx)
	}
}

func (c *coordinator) CreateMiddlewareWrapper(handler func(*Context) error) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(echoCtx echo.Context) error {
			ctx := c.acquireContext(echoCtx, "middleware", "handler")
			defer c.releaseContext(ctx)

			if err := handler(ctx); err != nil {
				return ctx.JSONError(err)
			}

			return next(echoCtx)
		}
	}
}
