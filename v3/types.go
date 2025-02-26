package raptor

import (
	"github.com/go-raptor/errs"
	"github.com/go-raptor/raptor/v3/components"
	"github.com/go-raptor/raptor/v3/config"
	"github.com/go-raptor/raptor/v3/core"
)

type Components = core.Components
type Config = config.Config
type Controller = components.Controller
type Controllers = components.Controllers
type Service = components.Service
type Services = components.Services
type Middleware = components.Middleware
type Middlewares = components.Middlewares
type Context = components.Context
type Error = errs.Error
type Utils = components.Utils
type Map = map[string]interface{}
