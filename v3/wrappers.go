package raptor

import "github.com/labstack/echo/v4"

func (r *Raptor) CreateActionWrapper(controller, action string, handler func(*Context) error) echo.HandlerFunc {
	return func(c echo.Context) error {
		ctx := r.AcquireContext(c, controller, action)
		defer r.ReleaseContext(ctx)
		return handler(ctx)
	}
}

func (r *Raptor) CreateMiddlewareWrapper(handler func(*Context) error) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			ctx := r.AcquireContext(c, "middleware", "handler")
			defer r.ReleaseContext(ctx)
			return handler(ctx)
		}
	}
}
