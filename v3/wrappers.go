package raptor

import "github.com/labstack/echo/v4"

func (r *Raptor) CreateActionWrapper(controller, action string, handler func(*Context) error) echo.HandlerFunc {
	return func(c echo.Context) error {
		ctx := r.acquireContext(c, controller, action)
		defer r.releaseContext(ctx)

		for _, middleware := range r.middlewares {
			if middleware.ShouldRun(controller, action) {
				if err := middleware.New(ctx); err != nil {
					return ctx.JSONError(err)
				}
			}
		}

		return handler(ctx)
	}
}

func (r *Raptor) CreateMiddlewareWrapper(handler func(*Context) error) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			ctx := r.acquireContext(c, "middleware", "handler")
			defer r.releaseContext(ctx)

			if err := handler(ctx); err != nil {
				return ctx.JSONError(err)
			}

			return next(c)
		}
	}
}
