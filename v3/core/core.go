package core

import (
	"context"
	"log/slog"
	"net/http"
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
			return mw.New(state, func(nextState components.State) error {
				return currentChain(nextState)
			})
		}
	}

	return chain(ctx)
}

func (c *Core) logRequest(ctx *Context, startTime time.Time, err error) {
	attrs := []any{
		"ip", ctx.RealIP(),
		"method", ctx.Request().Method,
		"path", ctx.Request().URL.Path,
		"duration", time.Since(startTime).Milliseconds(),
	}

	var (
		logLevel slog.Level
		message  string
		status   int
	)

	if err == nil {
		logLevel = slog.LevelInfo
		message = "Request processed"
		attrs = append(attrs,
			"status", ctx.Response().Status,
			"handler", ActionDescriptor(ctx.Controller(), ctx.Action()),
		)
		c.utils.Log.Log(context.Background(), logLevel, message, attrs...)
		return
	}

	// Handle error case
	logLevel = slog.LevelError
	if raptorErr, ok := err.(*errs.Error); ok {
		status = raptorErr.Code
		if status == http.StatusNotFound {
			message = "Handler not found"
		} else {
			message = "Error while processing request"
		}
		attrs = append(attrs, "message", raptorErr.Message)
		errAttrs := raptorErr.AttrsToSlice()
		for i := 0; i < len(errAttrs); i += 2 {
			if i+1 < len(errAttrs) {
				key := errAttrs[i]
				keyExists := false
				for j := 0; j < len(attrs); j += 2 {
					if j+1 < len(attrs) && attrs[j] == key {
						keyExists = true
						break
					}
				}
				if !keyExists {
					attrs = append(attrs, errAttrs[i], errAttrs[i+1])
				}
			}
		}
	} else {
		status = http.StatusInternalServerError
	}

	attrs = append(attrs, "status", status)
	c.utils.Log.Log(context.Background(), logLevel, message, attrs...)
}

func (c *Core) RegisterErrorHandler(server *echo.Echo) {
	server.HTTPErrorHandler = func(err error, ctx echo.Context) {
		if raptorError, ok := err.(*errs.Error); ok {
			ctx.JSON(raptorError.Code, raptorError)
		} else {
			ctx.JSON(http.StatusInternalServerError, errs.NewError(http.StatusInternalServerError, err.Error()))
		}
	}
}
