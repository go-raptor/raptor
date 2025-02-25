package core

import (
	"github.com/go-raptor/errs"
	"github.com/labstack/echo/v4"
)

func (c *Core) CreateActionWrapper(controller, action string, handler func(*Context) error) echo.HandlerFunc {
	return func(echoCtx echo.Context) error {
		ctx := c.acquireContext(echoCtx, controller, action)
		defer c.releaseContext(ctx)

		chain := handler
		for i := len(c.handlers[controller][action].middlewares) - 1; i >= 0; i-- {
			mwIndex := c.handlers[controller][action].middlewares[i]
			mw := c.middlewares[mwIndex]
			currentChain := chain
			chain = func(ctx *Context) error {
				if err := mw.New(ctx, currentChain); err != nil {
					if _, ok := err.(*errs.Error); ok {
						ctx.JSONError(err)
						return nil
					}
					return err
				}
				return nil
			}
		}

		return chain(ctx)
	}
}
