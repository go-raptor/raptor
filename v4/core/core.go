package core

import (
	"net/http"
	"sync"

	"github.com/go-raptor/raptor/v4/errs"
)

type Core struct {
	Resources   *Resources
	Handlers    map[string]map[string]*Handler
	Services    map[string]ServiceInitializer
	Middlewares []MiddlewareInitializer

	contextPool *sync.Pool
	Binder      Binder
}

func NewCore(resources *Resources) *Core {
	binder := &DefaultBinder{}

	core := &Core{
		Resources: resources,
		Handlers:  make(map[string]map[string]*Handler),
		Services:  make(map[string]ServiceInitializer),
		Binder:    binder,
	}
	core.contextPool = &sync.Pool{
		New: func() interface{} {
			return NewContext(core, nil, nil)
		},
	}
	return core
}

func (c *Core) Handler(w http.ResponseWriter, r *http.Request, controller, action string) {
	ctx := c.contextPool.Get().(*Context)
	ctx.Reset(r, w, controller, action)
	defer func() {
		c.contextPool.Put(ctx)
	}()

	handler, exists := c.Handlers[controller][action]
	if !exists {
		ctx.Error(errs.NewErrorInternal("Handler not found"))
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
			return mw.Handle(ctx, currentChain)
		}
	}

	err := chain(ctx)
	if err != nil {
		ctx.Error(err)
	}
}
