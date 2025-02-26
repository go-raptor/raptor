package core

import (
	"sync"

	"github.com/go-raptor/connector"
	"github.com/go-raptor/raptor/v3/components"
	"github.com/labstack/echo/v4"
)

type Components struct {
	DatabaseConnector connector.DatabaseConnector
	Middlewares       components.Middlewares
	Services          components.Services
	Controllers       components.Controllers
}

type Core struct {
	utils       *components.Utils
	handlers    map[string]map[string]*handler
	contextPool sync.Pool
	services    map[string]components.ServiceInterface
	middlewares []components.MiddlewareInterface
}

func NewCore(u *components.Utils) *Core {
	return &Core{
		utils:    u,
		handlers: make(map[string]map[string]*handler),
		contextPool: sync.Pool{
			New: func() interface{} {
				return new(components.Context)
			},
		},
		services:    make(map[string]components.ServiceInterface),
		middlewares: make([]components.MiddlewareInterface, 0),
	}
}

type echoMiddleware struct {
	middleware echo.MiddlewareFunc
	utils      *components.Utils
}
