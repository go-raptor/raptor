package core

import (
	"context"
	"net/http"
	"time"

	"github.com/go-raptor/raptor/v4/errs"
)

// readyPingTimeout bounds the database check of a readiness probe.
const readyPingTimeout = 2 * time.Second

var healthOK = []byte(`{"status":"ok"}`)

// HealthController answers liveness and readiness probes. Raptor registers
// it unless the app has its own HealthController; route it yourself, e.g.
// GET /healthz → Health.Live and GET /readyz → Health.Ready, and exclude it
// from authentication middleware.
type HealthController struct {
	Controller
}

// Live reports that the process serves requests at all.
func (h *HealthController) Live(ctx *Context) error {
	return ctx.JSONBlob(http.StatusOK, healthOK)
}

// Ready reports whether the app should get traffic: not while shutting
// down, and not while a database whose connector can Ping is unreachable.
func (h *HealthController) Ready(ctx *Context) error {
	if h.ShuttingDown() {
		return errs.NewErrorServiceUnavailable("Shutting down")
	}
	if pinger, ok := h.Database.(interface{ Ping(context.Context) error }); ok {
		pingCtx, cancel := context.WithTimeout(ctx.Request().Context(), readyPingTimeout)
		defer cancel()
		if err := pinger.Ping(pingCtx); err != nil {
			h.Log.Warn("Readiness check failed", "error", err)
			return errs.NewErrorServiceUnavailable("Database unavailable")
		}
	}
	return ctx.JSONBlob(http.StatusOK, healthOK)
}
