package raptor

type Services []ServiceInterface

type ServiceInterface interface {
	Init(u *Utils, s map[string]ServiceInterface)
}

type Service struct {
	*Utils
	Services map[string]ServiceInterface
	onInit   func()
}

func (s *Service) Init(u *Utils, svcs map[string]ServiceInterface) {
	s.Utils = u
	s.Services = svcs
	if s.onInit != nil {
		s.onInit()
	}
}

func (s *Service) OnInit(callback func()) {
	s.onInit = callback
}
