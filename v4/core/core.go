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
	trusted, err := TrustedProxies(resources.Config.ServerConfig.TrustedProxies)
	if err != nil {
		resources.Log.Error("Invalid trusted_proxies configuration", "error", err)
		panic(err)
	}
	switch strings.ToLower(resources.Config.ServerConfig.IPExtractor) {
	case "x-forwarded-for":
		core.IPExtractor = ExtractIPFromXFFHeader(trusted)
	case "x-real-ip":
		core.IPExtractor = ExtractIPFromRealIPHeader(trusted)
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

// BeginShutdown marks the app as shutting down; Raptor.Shutdown calls it
// first, so readiness fails while requests drain.
func (c *Core) BeginShutdown() {
	c.Resources.shuttingDown.Store(true)
}

// CancelAppContext cancels Resources.AppContext; Raptor.Shutdown calls it
// once requests have drained, before services clean up.
func (c *Core) CancelAppContext() {
	CancelAppContext(c.Resources)
}

// Serve dispatches a request through h's precompiled middleware chain.
func (c *Core) Serve(w http.ResponseWriter, r *http.Request, h *Handler, controller, action, path string, store map[string]any) {
	if max := c.Resources.Config.ServerConfig.MaxBodyBytes; max > 0 && r.Body != nil && r.Body != http.NoBody {
		r.Body = http.MaxBytesReader(w, r.Body, max)
	}

	ctx := c.contextPool.Get().(*Context)
	ctx.ResetAndInit(r, w, controller, action, path, store)
	defer c.finishRequest(ctx, r)

	if err := h.chain(ctx); err != nil {
		ctx.Error(err)
	}
}

func (c *Core) finishRequest(ctx *Context, original *http.Request) {
	rec := recover()
	removeSwappedMultipart(ctx.request, original)
	if rec != nil {
		if err, ok := rec.(error); ok && errors.Is(err, http.ErrAbortHandler) {
			c.releaseContext(ctx)
			panic(rec)
		}
		c.Resources.Log.Error("Panic recovered in handler", ctx.logAttrs("panic", rec, "stack", string(debug.Stack()))...)
		if !ctx.response.Committed {
			ctx.Error(errs.NewErrorInternal("Internal Server Error"))
		}
	}
	c.releaseContext(ctx)
}

func (c *Core) releaseContext(ctx *Context) {
	ctx.release()
	c.contextPool.Put(ctx)
}

// removeSwappedMultipart deletes the temp files of a multipart form parsed
// on a request a middleware substituted (r.WithContext and the like):
// net/http cleans up only the form on the request it passed in.
func removeSwappedMultipart(current, original *http.Request) {
	if current == nil || current == original || current.MultipartForm == nil || current.MultipartForm == original.MultipartForm {
		return
	}
	current.MultipartForm.RemoveAll()
}
