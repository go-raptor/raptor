package core

import (
	"log/slog"
	"time"

	"github.com/go-raptor/components"
)

type handler struct {
	action      func(*components.Context) error
	middlewares []uint8
}

func newHandler(action func(*components.Context) error) *handler {
	return &handler{
		action:      action,
		middlewares: make([]uint8, 0),
	}
}

func (h *handler) injectMiddleware(middlewareIndex uint8) {
	h.middlewares = append(h.middlewares, uint8(middlewareIndex))
}

func (c *Core) Handle(ctx *components.Context) error {
	startTime := time.Now()
	err := c.handlers[ctx.Controller][ctx.Action].action(ctx)
	if c.utils.LogLevel.Level() < slog.LevelWarn {
		c.logAction(ctx, startTime)
	}
	return err
}

func (c *Core) logAction(ctx *components.Context, startTime time.Time) {
	c.utils.Log.Info("Request processed",
		"ip", ctx.RealIP(),
		"method", ctx.Request().Method,
		"path", ctx.Request().URL.Path,
		"status", ctx.Response().Status,
		"handler", ActionDescriptor(ctx.Controller, ctx.Action),
		"duration", time.Since(startTime).Milliseconds(),
	)
}
