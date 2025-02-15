package core

import (
	"errors"
	"net/http"

	"github.com/labstack/echo/v4"
)

type Map map[string]interface{}

type Context struct {
	echo.Context
	Controller string
	Action     string
}

func (c *Context) JSON(data interface{}, status ...int) error {
	if len(status) == 0 {
		status = append(status, http.StatusOK)
	}
	return c.Context.JSON(status[0], data)
}

func (c *Context) JSONError(err error, status ...int) error {
	var e *Error
	if errors.As(err, &e) {
		return c.JSON(e, e.Code)
	}

	if len(status) == 0 {
		status = append(status, http.StatusInternalServerError)
	}
	return c.JSON(NewError(status[0], err.Error()), status[0])
}

func (c *Core) acquireContext(ec echo.Context, controller, action string) *Context {
	ctx := c.contextPool.Get().(*Context)
	ctx.Context = ec
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
