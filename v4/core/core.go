package core

import (
	"net/http"
	"sync"
)

type Core struct {
	Utils       *Utils
	Handlers    map[string]map[string]*Handler
	ContextPool *sync.Pool
}

func NewCore(utils *Utils) *Core {
	return &Core{
		Utils:    utils,
		Handlers: make(map[string]map[string]*Handler),
		ContextPool: &sync.Pool{
			New: func() interface{} {
				return &Context{}
			},
		},
	}
}

func (c *Core) RegisterHandler(controller, action string, handler func(*Context) error) {
	if c.Handlers[controller] == nil {
		c.Handlers[controller] = make(map[string]*Handler)
	}
	c.Handlers[controller][action] = &Handler{
		Action: handler,
	}
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
