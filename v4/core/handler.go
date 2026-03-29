package core

import (
	"net/http"
	"slices"
)

type Handler struct {
	Action      HandlerFunc
	middlewares []int
}

type HandlerFunc func(ctx *Context) error

func (h *Handler) injectMiddleware(middlewareIndex int) {
	if !slices.Contains(h.middlewares, middlewareIndex) {
		h.middlewares = append(h.middlewares, middlewareIndex)
	}
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
