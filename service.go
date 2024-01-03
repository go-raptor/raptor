package raptor

type ServiceInterface interface {
	SetUtils(u **Utils)
}

type Service struct {
	Utils **Utils
	Util  *Utils
}

func (s *Service) SetUtils(u **Utils) {
	s.Utils = u
	s.Util = *s.Utils
}

func (s *Service) GetUtils() *Utils {
	return *s.Utils
}
