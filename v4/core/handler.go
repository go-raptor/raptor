package core

import "net/http"

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

func (c *Core) Handle(w http.ResponseWriter, r *http.Request, controller, action string) error {
	ctx := c.ContextPool.Get().(*Context)
	ctx.Reset(r, w, controller, action)
	defer func() {
		c.ContextPool.Put(ctx)
	}()

	handler, exists := c.Handlers[controller][action]
	if !exists {
		http.Error(w, "Handler not found", http.StatusNotFound)
		return nil
	}

	chain := handler.Action
	for i := len(handler.middlewares) - 1; i >= 0; i-- {
		mwIndex := handler.middlewares[i]
		if mwIndex >= len(c.Middlewares) {
			c.Resources.Log.Error("Invalid middleware index", "index", mwIndex)
			continue
		}
		mw := c.Middlewares[mwIndex]
		currentChain := chain
		chain = func(ctx *Context) error {
			return mw.New(ctx, currentChain)
		}
	}

	return chain(ctx)
}
