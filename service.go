package raptor

type ServiceInterface interface {
	SetUtils(u *Utils)
}

type Services []ServiceInterface

type Service struct {
	Utils *Utils
}

func (s *Service) SetUtils(u *Utils) {
	s.Utils = u
}
