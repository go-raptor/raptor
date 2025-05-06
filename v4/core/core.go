package core

import (
	"net/http"
	"sync"
)

type Core struct {
	Resources   *Resources
	Handlers    map[string]map[string]*Handler
	Services    map[string]ServiceInitializer
	Middlewares []MiddlewareInitializer
	ContextPool *sync.Pool
}

func NewCore(resources *Resources) *Core {
	binder := &DefaultBinder{}

	return &Core{
		Resources: resources,
		Handlers:  make(map[string]map[string]*Handler),
		Services:  make(map[string]ServiceInitializer),
		ContextPool: &sync.Pool{
			New: func() interface{} {
				return NewContext(nil, nil, binder)
			},
		},
	}
}

func (c *Core) Handler(w http.ResponseWriter, r *http.Request, controller, action string) error {
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
