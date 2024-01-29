package raptor

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
