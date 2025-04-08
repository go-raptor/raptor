package core

import (
	"time"

	"github.com/go-raptor/components"
	"github.com/go-raptor/connectors"
	"github.com/go-raptor/errs"
	"github.com/labstack/echo/v4"
)

type Components struct {
	DatabaseConnector connectors.DatabaseConnector
	Controllers       components.Controllers
	Services          components.Services
	Middlewares       components.Middlewares
}

type Core struct {
	utils       *components.Utils
	handlers    map[string]map[string]*handler
	services    map[string]components.ServiceProvider
	middlewares []components.MiddlewareProvider
}

func NewCore(u *components.Utils) *Core {
	return &Core{
		utils:       u,
		handlers:    make(map[string]map[string]*handler),
		services:    make(map[string]components.ServiceProvider),
		middlewares: make([]components.MiddlewareProvider, 0),
	}
}

func (c *Core) Handle(echoCtx echo.Context) error {
	ctx := GetContext(echoCtx)

	h := c.handlers[ctx.Controller()][ctx.Action()]
	chain := h.action

	for i := len(c.handlers[ctx.Controller()][ctx.Action()].middlewares) - 1; i >= 0; i-- {
		mwIndex := c.handlers[ctx.Controller()][ctx.Action()].middlewares[i]
		mw := c.middlewares[mwIndex]
		currentChain := chain
		chain = func(state components.State) error {
			if err := mw.New(state, func(nextState components.State) error {
				return currentChain(nextState)
			}); err != nil {
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

func (c *Core) logRequest(ctx *Context, startTime time.Time) {
	c.utils.Log.Info("Request processed",
		"ip", ctx.RealIP(),
		"method", ctx.Request().Method,
		"path", ctx.Request().URL.Path,
		"status", ctx.Response().Status,
		"handler", ActionDescriptor(ctx.Controller(), ctx.Action()),
		"duration", time.Since(startTime).Milliseconds(),
	)
}
