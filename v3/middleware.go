package raptor

import (
	"github.com/labstack/echo/v4"
)

type Middlewares []MiddlewareInterface

type MiddlewareInterface interface {
	InitMiddleware(r *Raptor)
	New(*Context) error
	ShouldRun(controller, action string) bool
}

type Middleware struct {
	*Utils
	onInit    func()
	onlyMap   map[string]struct{}
	exceptMap map[string]struct{}
	scoped    bool
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

func (m *Middleware) ShouldRun(controller, action string) bool {
	if !m.scoped {
		return true
	}

	route := controller + "#" + action

	if m.exceptMap != nil {
		if _, exists := m.exceptMap[route]; exists {
			return false
		}
	}

	if m.onlyMap != nil && len(m.onlyMap) > 0 {
		_, exists := m.onlyMap[route]
		return exists
	}

	return true
}

func (m *Middleware) Only(only ...string) {
	m.scoped = true
	if m.onlyMap == nil {
		m.onlyMap = make(map[string]struct{})
	}

	for _, route := range only {
		m.onlyMap[route] = struct{}{}
	}
}

func (m *Middleware) Except(except ...string) {
	m.scoped = true
	if m.exceptMap == nil {
		m.exceptMap = make(map[string]struct{})
	}

	for _, route := range except {
		m.exceptMap[route] = struct{}{}
	}
}

type echoMiddleware struct {
	handler echo.HandlerFunc
}

func (m *echoMiddleware) New(c *Context) error {
	return m.handler(c.Context)
}

func (m *echoMiddleware) InitMiddleware(r *Raptor) {
}

func (m *echoMiddleware) ShouldRun(controller, action string) bool {
	return true
}

func Use(h echo.HandlerFunc) *echoMiddleware {
	return &echoMiddleware{handler: h}
}
