package core

import (
	"net/http"
	"strings"
	"sync"
)

type Core struct {
	Resources   *Resources
	Handlers    map[string]map[string]*Handler
	Services    map[string]ServiceInitializer
	Middlewares []MiddlewareInitializer

	contextPool *sync.Pool
	IPExtractor IPExtractor
}

func NewCore(resources *Resources) *Core {
	core := &Core{
		Resources: resources,
		Handlers:  make(map[string]map[string]*Handler),
		Services:  make(map[string]ServiceInitializer),
	}
	core.contextPool = &sync.Pool{
		New: func() any {
			return NewContext(core, nil, nil)
		},
	}
	switch strings.ToLower(resources.Config.ServerConfig.IPExtractor) {
	case "x-forwarded-for":
		core.IPExtractor = ExtractIPFromXFFHeader()
	case "x-real-ip":
		core.IPExtractor = ExtractIPFromRealIPHeader()
	default:
		core.IPExtractor = ExtractIPDirect()
	}
	return core
}

func (c *Core) Handler(w http.ResponseWriter, r *http.Request, controller, action, path string, store map[string]any) {
	ctx := c.contextPool.Get().(*Context)
	ctx.ResetAndInit(r, w, controller, action, path, store)
	defer c.contextPool.Put(ctx)

	handler := c.Handlers[controller][action]
	chain := handler.Action
	for i := len(handler.middlewares) - 1; i >= 0; i-- {
		mw := c.Middlewares[handler.middlewares[i]]
		next := chain
		chain = func(ctx *Context) error {
			return mw.Handle(ctx, next)
		}
	}

	if err := chain(ctx); err != nil {
		ctx.Error(err)
	}
}
