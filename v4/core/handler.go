package core

import (
	"net/http"
	"slices"
)

type Handler struct {
	Action      HandlerFunc
	middlewares []int
	chain       HandlerFunc
}

type HandlerFunc func(ctx *Context) error

func (h *Handler) injectMiddleware(middlewareIndex int) {
	if !slices.Contains(h.middlewares, middlewareIndex) {
		h.middlewares = append(h.middlewares, middlewareIndex)
	}
}

func (h *Handler) compile(middlewares []MiddlewareInitializer) {
	chain := h.Action
	for i := len(h.middlewares) - 1; i >= 0; i-- {
		mw := middlewares[h.middlewares[i]]
		next := chain
		chain = func(ctx *Context) error {
			return mw.Handle(ctx, next)
		}
	}
	h.chain = chain
}

func WrapHandler(h http.Handler) HandlerFunc {
	return func(ctx *Context) error {
		h.ServeHTTP(ctx.Response(), ctx.Request())
		return nil
	}
}

func WrapHandlerFunc(h http.HandlerFunc) HandlerFunc {
	return func(ctx *Context) error {
		h(ctx.Response(), ctx.Request())
		return nil
	}
}
