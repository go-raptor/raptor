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
	chain := wrapErr(h.Action)
	for i := len(h.middlewares) - 1; i >= 0; i-- {
		mw := middlewares[h.middlewares[i]]
		next := chain
		chain = wrapErr(func(ctx *Context) error {
			return mw.Handle(ctx, next)
		})
	}
	h.chain = chain
}

// wrapErr commits the error response at the layer where the error occurred,
// so outer middleware (loggers, metrics) observe the final status after
// next() returns. The tradeoff: outer middleware cannot replace an error
// response an inner layer has already written.
func wrapErr(fn HandlerFunc) HandlerFunc {
	return func(ctx *Context) error {
		err := fn(ctx)
		if err != nil && !ctx.Response().Committed {
			ctx.Error(err)
		}
		return err
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
