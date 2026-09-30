package raptor_test

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"runtime"
	"strings"
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
		{"[::1]", "::1"},               // bracketed IPv6
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

type EchoController struct{ raptor.Controller }

func (c *EchoController) Echo(ctx *raptor.Context) error {
	var body map[string]string
	if err := ctx.Bind(&body); err != nil {
		return err
	}
	cookie, _ := ctx.Cookie("session")
	if cookie != nil {
		body["session"] = cookie.Value
	}
	return ctx.Data(body, http.StatusCreated)
}

func TestJSONHelpers(t *testing.T) {
	app := raptor.NewTestApp(
		&raptor.Components{Controllers: raptor.Controllers{&EchoController{}}},
		router.CollectRoutes(router.Post("/echo", "Echo.Echo")),
	)
	rec := app.TestPost("/echo", raptor.JSONBody(t, map[string]string{"name": "raptor"}),
		raptor.WithCookie(&http.Cookie{Name: "session", Value: "s1"}))
	got := raptor.DecodeJSON[map[string]string](t, rec, http.StatusCreated)
	if got["name"] != "raptor" || got["session"] != "s1" {
		t.Fatalf("got %v", got)
	}
}

type recordingT struct{ failed string }

func (r *recordingT) Helper() {}
func (r *recordingT) Fatalf(format string, args ...any) {
	r.failed = fmt.Sprintf(format, args...)
	runtime.Goexit()
}

func TestDecodeJSONFailsOnWrongStatus(t *testing.T) {
	rec := httptest.NewRecorder()
	rec.WriteHeader(http.StatusTeapot)
	rt := &recordingT{}
	done := make(chan struct{})
	go func() {
		defer close(done)
		raptor.DecodeJSON[map[string]any](rt, rec, http.StatusOK)
	}()
	<-done
	if !strings.Contains(rt.failed, "418") {
		t.Fatalf("DecodeJSON must fail on an unexpected status, got %q", rt.failed)
	}
}
