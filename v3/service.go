package raptor

type Services []ServiceInterface

type ServiceInterface interface {
	InitService(r *Raptor) error
}

type Service struct {
	*Utils
	onInit func() error
}

func (s *Service) InitService(r *Raptor) error {
	s.Utils = r.Utils
	if s.onInit != nil {
		return s.onInit()
	}
	return nil
}

func (s *Service) OnInit(callback func() error) {
	s.onInit = callback
}
