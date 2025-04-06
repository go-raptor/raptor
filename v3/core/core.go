package core

import (
	"github.com/go-raptor/components"
	"github.com/go-raptor/connectors"
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
