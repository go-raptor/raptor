package raptor

type Services []ServiceInterface

type ServiceInterface interface {
	Init(u *Utils)
}

type Service struct {
	Utils  *Utils
	onInit func()
}

func (s *Service) Init(u *Utils) {
	s.Utils = u
	if s.onInit != nil {
		s.onInit()
	}
}

func (s *Service) OnInit(callback func()) {
	s.onInit = callback
}
