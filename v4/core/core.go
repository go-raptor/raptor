package core

import (
	"errors"
	"net/http"
	"runtime/debug"
	"strings"
	"sync"

	"github.com/go-raptor/raptor/v4/errs"
)

type Core struct {
	Resources   *Resources
	Handlers    map[string]map[string]*Handler
	Services    map[string]ServiceInitializer
	Middlewares []MiddlewareInitializer

	serviceOrder []string
	contextPool  *sync.Pool
	IPExtractor  IPExtractor
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

// CompileHandlers builds each handler's middleware chain. Must be called
// after all middlewares are registered and before serving.
func (c *Core) CompileHandlers() {
	for _, actions := range c.Handlers {
		for _, h := range actions {
			h.compile(c.Middlewares)
		}
	}
}

// Serve dispatches a request through h's precompiled middleware chain.
func (c *Core) Serve(w http.ResponseWriter, r *http.Request, h *Handler, controller, action, path string, store map[string]any) {
	if max := c.Resources.Config.ServerConfig.MaxBodyBytes; max > 0 && r.Body != nil && r.Body != http.NoBody {
		r.Body = http.MaxBytesReader(w, r.Body, max)
	}

	ctx := c.contextPool.Get().(*Context)
	ctx.ResetAndInit(r, w, controller, action, path, store)
	defer c.finishRequest(ctx)

	if err := h.chain(ctx); err != nil {
		ctx.Error(err)
	}
}

func (c *Core) finishRequest(ctx *Context) {
	if rec := recover(); rec != nil {
		if err, ok := rec.(error); ok && errors.Is(err, http.ErrAbortHandler) {
			c.contextPool.Put(ctx)
			panic(rec)
		}
		c.Resources.Log.Error("Panic recovered in handler", "controller", ctx.controller, "action", ctx.action, "panic", rec, "stack", string(debug.Stack()))
		if !ctx.response.Committed {
			ctx.Error(errs.NewErrorInternal("Internal Server Error"))
		}
	}
	c.contextPool.Put(ctx)
}
