package raptor

import "github.com/gofiber/fiber/v2"

type Middlewares []MiddlewareInterface

type MiddlewareInterface interface {
	Init(u *Utils)
	New(*Context) error
}

type Middleware struct {
	Utils  *Utils
	onInit func()
}

func (m *Middleware) Init(u *Utils) {
	m.Utils = u
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
	return m.handler(c.Ctx)
}

func (m *fiberMiddleware) Init(u *Utils) {
}

func Use(h fiber.Handler) *fiberMiddleware {
	return &fiberMiddleware{handler: h}
}
