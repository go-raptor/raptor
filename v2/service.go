package raptor

type Services []ServiceInterface

type ServiceInterface interface {
	InitService(u *Utils, r *Routes)
}

type Service struct {
	*Utils
	Routes *Routes
	onInit func()
}

func (s *Service) InitService(u *Utils, r *Routes) {
	s.Utils = u
	s.Routes = r
	if s.onInit != nil {
		s.onInit()
	}
}

func (s *Service) OnInit(callback func()) {
	s.onInit = callback
}
