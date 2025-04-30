package core

import "net/http"

type Handler struct {
	Action func(*Context) error
}

func (c *Core) Handle(w http.ResponseWriter, r *http.Request, controller, action string) error {
	ctx := c.ContextPool.Get().(*Context)
	defer func() {
		ctx.Reset()
		c.ContextPool.Put(ctx)
	}()

	ctx.Writer = w
	ctx.Request = r
	ctx.Controller = controller
	ctx.Action = action

	handler, exists := c.Handlers[controller][action]
	if !exists {
		http.Error(w, "Handler not found", http.StatusNotFound)
		return nil
	}

	chain := handler.Action
	/*for i := len(handler.Middlewares) - 1; i >= 0; i-- {
		mwIndex := handler.Middlewares[i]
		if mwIndex >= len(c.Middlewares) {
			c.Utils.Log.Error("Invalid middleware index", "index", mwIndex)
			continue
		}
		mw := c.Middlewares[mwIndex]
		currentChain := chain
		chain = func(ctx *Context) error {
			return mw(ctx, currentChain)
		}
	}*/

	return chain(ctx)
}
