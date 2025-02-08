package raptor

import (
	"github.com/labstack/echo/v4"
)

type ScopedMiddleware struct {
	middleware MiddlewareInterface
	scope      string
}
type Middlewares []ScopedMiddleware

type MiddlewareInterface interface {
	InitMiddleware(r *Raptor)
	New(*Context) error
}

type Middleware struct {
	*Utils
	onInit func()
}

func (m *Middleware) InitMiddleware(r *Raptor) {
	m.Utils = r.Utils
	if m.onInit != nil {
		m.onInit()
	}
}

func (m *Middleware) OnInit(callback func()) {
	m.onInit = callback
}

type echoMiddleware struct {
	handler echo.HandlerFunc
}

func (m *echoMiddleware) New(c *Context) error {
	return m.handler(c.Context)
}

func (m *echoMiddleware) InitMiddleware(r *Raptor) {
}

func UseEcho(h echo.HandlerFunc) *echoMiddleware {
	return &echoMiddleware{handler: h}
}

func Use(middleware MiddlewareInterface) ScopedMiddleware {
	return ScopedMiddleware{
		middleware: middleware,
		scope:      "*",
	}
}
