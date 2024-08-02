package raptor

import "github.com/gofiber/fiber/v3"

type Middlewares []MiddlewareInterface

type MiddlewareInterface interface {
	InitMiddleware(r *Raptor)
	New(*Context) error
}

type Middleware struct {
	*Utils
	*Raptor
	onInit func()
}

func (m *Middleware) InitMiddleware(r *Raptor) {
	m.Utils = r.Utils
	m.Raptor = r
	if m.onInit != nil {
		m.onInit()
	}
}

func (m *Middleware) OnInit(callback func()) {
	m.onInit = callback
}

type fiberMiddleware struct {
	handler fiber.Handler
}

func (m *fiberMiddleware) New(c *Context) error {
	return m.handler(c.DefaultCtx)
}

func (m *fiberMiddleware) InitMiddleware(r *Raptor) {
}

func Use(h fiber.Handler) *fiberMiddleware {
	return &fiberMiddleware{handler: h}
}
