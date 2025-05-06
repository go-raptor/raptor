package core

type Handler struct {
	Action      HandlerFunc
	middlewares []int
}

type HandlerFunc func(ctx *Context) error

func (h *Handler) injectMiddleware(middlewareIndex int) {
	for _, idx := range h.middlewares {
		if idx == middlewareIndex {
			return
		}
	}
	h.middlewares = append(h.middlewares, middlewareIndex)
}
