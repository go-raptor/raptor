package core

type Handler struct {
	Action func(*Context) error
}
