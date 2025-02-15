package core

import (
	"fmt"
	"sync"

	"github.com/go-raptor/raptor/v3/router"
	"github.com/labstack/echo/v4"
)

type Core struct {
	utils       *Utils
	handlers    map[string]map[string]*handler
	contextPool sync.Pool
	services    map[string]ServiceInterface
	middlewares []MiddlewareInterface
	routes      router.Routes
}

func NewCore(u *Utils) *Core {
	return &Core{
		utils:    u,
		handlers: make(map[string]map[string]*handler),
		contextPool: sync.Pool{
			New: func() interface{} {
				return new(Context)
			},
		},
		services:    make(map[string]ServiceInterface),
		middlewares: make([]MiddlewareInterface, 0),
	}
}

func (c *Core) RegisterRoutes(app *App, server *echo.Echo) error {
	c.routes = app.Routes
	for _, route := range c.routes {
		if !c.hasControllerAction(route.Controller, route.Action) {
			err := fmt.Errorf("action %s not found in controller %s for path %s", route.Action, route.Controller, route.Path)
			c.utils.Log.Error("Error while registering route", "error", err)
			return err
		}
		c.registerRoute(route, server)
	}

	return nil
}

func (c *Core) registerRoute(route router.Route, server *echo.Echo) {
	routeHandler := c.CreateActionWrapper(route.Controller, route.Action, c.handle)
	if route.Method != "*" {
		server.Add(route.Method, route.Path, routeHandler)
		return
	}
	server.Any(route.Path, routeHandler)
}
