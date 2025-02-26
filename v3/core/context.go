package core

import (
	"github.com/go-raptor/raptor/v3/components"
	"github.com/labstack/echo/v4"
)

func (c *Core) acquireContext(echoContext echo.Context, controller, action string) *components.Context {
	ctx := c.contextPool.Get().(*components.Context)
	ctx.Context = echoContext
	ctx.Controller = controller
	ctx.Action = action
	return ctx
}

func (c *Core) releaseContext(ctx *components.Context) {
	if ctx == nil {
		return
	}
	ctx.Context = nil
	ctx.Controller = ""
	ctx.Action = ""
	c.contextPool.Put(ctx)
}
