package components

type Services []ServiceInterface

type ServiceInterface interface {
	InitService(u *Utils) error
}

type Service struct {
	*Utils
	onInit func() error
}

func (s *Service) InitService(utils *Utils) error {
	s.Utils = utils
	if s.onInit != nil {
		return s.onInit()
	}
	return nil
}

func (s *Service) OnInit(callback func() error) {
	s.onInit = callback
}
