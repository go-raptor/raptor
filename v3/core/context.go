package core

import (
	"net/http"

	"github.com/labstack/echo/v4"
)

type ContextKey string

const (
	ControllerKey ContextKey = "controller"
	ActionKey     ContextKey = "action"
)

type Context struct {
	echo.Context
}

func NewContext(c echo.Context) *Context {
	return &Context{Context: c}
}

func GetContext(c echo.Context) *Context {
	if ctx, ok := c.(*Context); ok {
		return ctx
	}
	return NewContext(c)
}

func (c *Context) Controller() string {
	if v := c.Get(string(ControllerKey)); v != nil {
		return v.(string)
	}
	return ""
}

func (c *Context) Action() string {
	if v := c.Get(string(ActionKey)); v != nil {
		return v.(string)
	}
	return ""
}

func (c *Context) JSONResponse(data interface{}, status ...int) error {
	if len(status) == 0 {
		status = append(status, http.StatusOK)
	}
	c.Context.JSON(status[0], data)
	return nil
}

func (c *Context) ResponseWriter() http.ResponseWriter {
	return c.Context.Response().Writer
}
