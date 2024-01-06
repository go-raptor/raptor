package raptor

type Services []ServiceInterface

type ServiceInterface interface {
	SetUtils(u *Utils)
}

type Service struct {
	Utils *Utils
}

func (s *Service) SetUtils(u *Utils) {
	s.Utils = u
}
