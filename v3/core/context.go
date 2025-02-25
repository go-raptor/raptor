package core

import (
	"errors"
	"net/http"

	"github.com/labstack/echo/v4"
)

func (c *Context) JSON(data interface{}, status ...int) error {
	if len(status) == 0 {
		status = append(status, http.StatusOK)
	}
	return c.Context.JSON(status[0], data)
}

func (c *Context) JSONError(err error, status ...int) error {
	var e *Error
	if errors.As(err, &e) {
		c.JSON(e, e.Code)
		return nil
	}

	if len(status) == 0 {
		status = append(status, http.StatusInternalServerError)
	}
	c.JSON(NewError(status[0], err.Error()), status[0])
	return nil
}

func (c *Core) acquireContext(echoContext echo.Context, controller, action string) *Context {
	ctx := c.contextPool.Get().(*Context)
	ctx.Context = echoContext
	ctx.Controller = controller
	ctx.Action = action
	return ctx
}

func (c *Core) releaseContext(ctx *Context) {
	if ctx == nil {
		return
	}
	ctx.Context = nil
	ctx.Controller = ""
	ctx.Action = ""
	c.contextPool.Put(ctx)
}
