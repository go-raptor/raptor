package raptor

import (
	"net/http"

	"github.com/go-raptor/raptor/v4/config"
	"github.com/go-raptor/raptor/v4/core"
)

type Context = core.Context
type Controller = core.Controller
type Controllers = core.Controllers
type Config = config.Config
type Components = core.Components
type Service = core.Service
type Services = core.Services
type Middleware = core.Middleware
type MiddlewareInitializer = core.MiddlewareInitializer
type Middlewares = core.Middlewares
type Resources = core.Resources
type HandlerFunc = core.HandlerFunc

var (
	WrapHandler     = core.WrapHandler
	WrapHandlerFunc = core.WrapHandlerFunc
	UseStd          = core.UseStd
	UseStdOnly      = core.UseStdOnly
	UseStdExcept    = core.UseStdExcept
)

func WrapMiddleware(mw func(http.Handler) http.Handler) core.ScopedMiddleware {
	return core.UseStd(mw)
}
