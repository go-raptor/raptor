package raptor

type ServiceInterface interface {
	SetUtils(u **Utils)
}

type Service struct {
	utils **Utils
}

func (s *Service) SetUtils(u **Utils) {
	s.utils = u
}

func (s *Service) Utils() *Utils {
	return *s.utils
}
