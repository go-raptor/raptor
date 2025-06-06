package core

import "github.com/go-raptor/components"

type handler struct {
	action      func(components.State) error
	middlewares []uint8
}

func newHandler(action func(components.State) error) *handler {
	return &handler{
		action:      action,
		middlewares: make([]uint8, 0),
	}
}

func (h *handler) injectMiddleware(middlewareIndex uint8) {
	h.middlewares = append(h.middlewares, uint8(middlewareIndex))
}
