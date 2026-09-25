package raptor_test

import (
	"net/http"
	"sync"
	"testing"

	"github.com/go-raptor/raptor/v4"
	"github.com/go-raptor/raptor/v4/config"
	"github.com/go-raptor/raptor/v4/errs"
	"github.com/go-raptor/raptor/v4/router"
)

type ClientIPController struct {
	raptor.Controller
}

func (c *ClientIPController) Show(ctx *raptor.Context) error {
	return ctx.String(http.StatusOK, ctx.RealIP())
}

// OncePerIPMiddleware admits each client IP once, like a rate limiter with burst 1.
type OncePerIPMiddleware struct {
	raptor.Middleware

	mu   sync.Mutex
	seen map[string]bool
}

func (m *OncePerIPMiddleware) Handle(ctx *raptor.Context, next func(*raptor.Context) error) error {
	m.mu.Lock()
	ip := ctx.RealIP()
	repeat := m.seen[ip]
	m.seen[ip] = true
	m.mu.Unlock()
	if repeat {
		return errs.NewErrorTooManyRequests("Rate limit exceeded")
	}
	return next(ctx)
}

func TestWithRemoteAddrGivesEachClientItsOwnIP(t *testing.T) {
	app := raptor.NewTestApp(&raptor.Components{
		Controllers: raptor.Controllers{&ClientIPController{}},
		Middlewares: raptor.Middlewares{raptor.Use(&OncePerIPMiddleware{seen: map[string]bool{}})},
	}, router.CollectRoutes(router.Get("/ip", "ClientIP.Show")))

	for _, tc := range []struct{ addr, want string }{
		{"10.0.0.1:5000", "10.0.0.1"},
		{"10.0.0.2", "10.0.0.2"},       // port optional
		{"2001:db8::1", "2001:db8::1"}, // bare IPv6
	} {
		rec := app.TestGet("/ip", raptor.WithRemoteAddr(tc.addr))
		if rec.Code != http.StatusOK || rec.Body.String() != tc.want {
			t.Fatalf("WithRemoteAddr(%q): got %d %q, want 200 %q", tc.addr, rec.Code, rec.Body, tc.want)
		}
	}
	if rec := app.TestGet("/ip", raptor.WithRemoteAddr("10.0.0.1:6000")); rec.Code != http.StatusTooManyRequests {
		t.Fatalf("the same IP must share its bucket: got %d, want 429", rec.Code)
	}
}

func TestNewTestAppAcceptsAppConfig(t *testing.T) {
	app := raptor.NewTestApp(&raptor.Components{}, router.Routes{},
		raptor.WithConfig(&config.Config{AppConfig: map[string]string{"feature": "on"}}))
	if got := app.Core.Resources.Config.AppConfig["feature"]; got != "on" {
		t.Fatalf("got %q, want on", got)
	}
}
