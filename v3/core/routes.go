package core

import (
	"fmt"

	"github.com/go-raptor/raptor/v3/router"
	"github.com/labstack/echo/v4"
)

func (c *Core) RegisterRoutes(components *Components, server *echo.Echo) error {
	c.routes = components.Routes
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
