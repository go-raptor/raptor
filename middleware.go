package raptor

type Middlewares []MiddlewareInterface

type MiddlewareInterface interface {
	SetUtils(u *Utils)
	New(*Context) error
}

type Middleware struct {
	Utils *Utils
}

func (m *Middleware) SetUtils(u *Utils) {
	m.Utils = u
}
