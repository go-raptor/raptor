package raptor

type handler struct {
	action      func(*Context) error
	middlewares []uint8
}
