package raptor

import "github.com/labstack/echo/v4"

func wrapMiddlewareHandler(handler func(*Context) error) echo.MiddlewareFunc {
	return echo.MiddlewareFunc(func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			context := &Context{c, "", ""}
			err := handler(context)
			if err != nil {
				return context.JSONError(err)
			}
			return next(c)
		}
	})
}

func wrapActionHandler(controller, action string, handler func(*Context) error) echo.HandlerFunc {
	return func(c echo.Context) error {
		return handler(&Context{c, controller, action})
	}
}
