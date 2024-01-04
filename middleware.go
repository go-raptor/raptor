package raptor

type MiddlewareInterface interface {
	SetUtils(u *Utils)
	New(*Context) error
}

type Middlewares []MiddlewareInterface

type Middleware struct {
	Utils *Utils
}

func (m *Middleware) SetUtils(u *Utils) {
	m.Utils = u
}
