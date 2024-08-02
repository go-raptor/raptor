package raptor

type Services []ServiceInterface

type ServiceInterface interface {
	InitService(r *Raptor)
}

type Service struct {
	*Utils
	*Raptor
	onInit func()
}

func (s *Service) InitService(r *Raptor) {
	s.Utils = r.Utils
	s.Raptor = r
	if s.onInit != nil {
		s.onInit()
	}
}

func (s *Service) OnInit(callback func()) {
	s.onInit = callback
}
