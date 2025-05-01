package components

type Services []ServiceInitializer

type ServiceInitializer interface {
	Init(*Resources) error
	Shutdown() error
}

type Service struct {
	*Resources
	onInit     func() error
	onShutdown func() error
}

func (s *Service) Init(resources *Resources) error {
	s.Resources = resources
	if s.onInit != nil {
		return s.onInit()
	}
	return nil
}

func (s *Service) Shutdown() error {
	if s.onShutdown != nil {
		return s.onShutdown()
	}
	return nil
}

func (s *Service) OnInit(callback func() error) {
	s.onInit = callback
}

func (s *Service) OnShutdown(callback func() error) {
	s.onShutdown = callback
}
