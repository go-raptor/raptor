package core

import (
	"log/slog"
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
	services    map[string]components.ServiceInterface
	middlewares []components.MiddlewareInterface
}

func NewCore(u *components.Utils) *Core {
	return &Core{
		utils:       u,
		handlers:    make(map[string]map[string]*handler),
		services:    make(map[string]components.ServiceInterface),
		middlewares: make([]components.MiddlewareInterface, 0),
	}
}

func (c *Core) Handle(echoCtx echo.Context) error {
	ctx := GetContext(echoCtx)
	startTime := time.Now()

	h := c.handlers[ctx.Controller()][ctx.Action()]
	chain := h.action

	for i := len(c.handlers[ctx.Controller()][ctx.Action()].middlewares) - 1; i >= 0; i-- {
		mwIndex := c.handlers[ctx.Controller()][ctx.Action()].middlewares[i]
		mw := c.middlewares[mwIndex]
		currentChain := chain
		chain = func(coreCtx *Context) error {
			if err := mw.New(coreCtx, func(ctxInt components.ContextInterface) error {
				return currentChain(coreCtx)
			}); err != nil {
				if _, ok := err.(*errs.Error); ok {
					coreCtx.JSONError(err)
					return nil
				}
				return err
			}
			return nil
		}
	}

	err := chain(ctx)

	if c.utils.LogLevel.Level() < slog.LevelWarn {
		c.logAction(ctx, startTime)
	}

	return err
}

func (c *Core) logAction(ctx *Context, startTime time.Time) {
	c.utils.Log.Info("Request processed",
		"ip", ctx.RealIP(),
		"method", ctx.Request().Method,
		"path", ctx.Request().URL.Path,
		"status", ctx.Response().Status,
		"handler", ActionDescriptor(ctx.Controller(), ctx.Action()),
		"duration", time.Since(startTime).Milliseconds(),
	)
}
