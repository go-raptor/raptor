package raptor_test

import (
	"context"
	"errors"
	"net/http"
	"testing"

	"github.com/go-raptor/connectors"
	"github.com/go-raptor/raptor/v4"
	"github.com/go-raptor/raptor/v4/config"
	"github.com/go-raptor/raptor/v4/router"
)

type pingConnector struct{ err error }

func (p *pingConnector) SetConfig(any)                  {}
func (p *pingConnector) Init() error                    { return nil }
func (p *pingConnector) Conn() any                      { return nil }
func (p *pingConnector) Migrator() connectors.Migrator  { return nil }
func (p *pingConnector) Ping(ctx context.Context) error { return p.err }

func newHealthApp(db *pingConnector) *raptor.Raptor {
	components := &raptor.Components{}
	var opts []raptor.RaptorOption
	if db != nil {
		components.DatabaseConnector = db
		opts = append(opts, raptor.WithConfig(&config.Config{DatabaseConfig: config.DatabaseConfig{Name: "test"}}))
	}
	return raptor.NewTestApp(components, router.CollectRoutes(
		router.Get("/healthz", "Health.Live"),
		router.Get("/readyz", "Health.Ready"),
	), opts...)
}

func TestHealthLiveAndReady(t *testing.T) {
	app := newHealthApp(&pingConnector{})
	for _, path := range []string{"/healthz", "/readyz"} {
		if rec := app.TestGet(path); rec.Code != http.StatusOK || rec.Body.String() != `{"status":"ok"}` {
			t.Errorf("%s: %d %s", path, rec.Code, rec.Body)
		}
	}
}

func TestReadyFailsWhenDatabaseDown(t *testing.T) {
	app := newHealthApp(&pingConnector{err: errors.New("connection refused")})
	if rec := app.TestGet("/readyz"); rec.Code != http.StatusServiceUnavailable {
		t.Fatalf("got %d, want 503", rec.Code)
	}
	if rec := app.TestGet("/healthz"); rec.Code != http.StatusOK {
		t.Fatalf("liveness must not depend on the database: %d", rec.Code)
	}
}

func TestReadyFailsOnceShutdownBegins(t *testing.T) {
	app := newHealthApp(nil)
	app.Core.BeginShutdown()
	if rec := app.TestGet("/readyz"); rec.Code != http.StatusServiceUnavailable {
		t.Fatalf("readiness during shutdown: got %d, want 503", rec.Code)
	}
	if rec := app.TestGet("/healthz"); rec.Code != http.StatusOK {
		t.Fatalf("liveness during shutdown: got %d, want 200", rec.Code)
	}
}

// An app's own HealthController wins; the built-in must not merge into it.
type HealthController struct{ raptor.Controller }

func (c *HealthController) Custom(ctx *raptor.Context) error {
	return ctx.String(http.StatusOK, "mine")
}

func TestAppHealthControllerWins(t *testing.T) {
	app := raptor.NewTestApp(
		&raptor.Components{Controllers: raptor.Controllers{&HealthController{}}},
		router.CollectRoutes(router.Get("/custom", "Health.Custom")),
	)
	if rec := app.TestGet("/custom"); rec.Body.String() != "mine" {
		t.Fatalf("got %q", rec.Body.String())
	}
	if _, ok := app.Core.Handlers["HealthController"]["Ready"]; ok {
		t.Fatal("the built-in actions must not merge into an app's HealthController")
	}
}
