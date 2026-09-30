package raptor

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"testing"
	"time"

	"github.com/go-raptor/connectors"
	"github.com/go-raptor/raptor/v4/config"
	"github.com/go-raptor/raptor/v4/router"
)

type closeRecorder struct{ closed bool }

func (c *closeRecorder) SetConfig(any)                 {}
func (c *closeRecorder) Init() error                   { return nil }
func (c *closeRecorder) Conn() any                     { return nil }
func (c *closeRecorder) Migrator() connectors.Migrator { return nil }
func (c *closeRecorder) Close() error                  { c.closed = true; return nil }

func newServeTestApp(t *testing.T, port int) (*Raptor, *closeRecorder) {
	t.Helper()
	db := &closeRecorder{}
	app := NewTestApp(&Components{DatabaseConnector: db}, nil, WithConfig(&config.Config{
		ServerConfig:   config.ServerConfig{Address: "127.0.0.1", Port: port},
		DatabaseConfig: config.DatabaseConfig{Name: "test"},
	}))
	return app, db
}

func TestServeShutsDownWhenContextEnds(t *testing.T) {
	probe, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	port := probe.Addr().(*net.TCPAddr).Port
	probe.Close()
	app, db := newServeTestApp(t, port)

	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if err := app.serve(ctx); err != nil {
		t.Fatalf("serve: %v", err)
	}
	if !db.closed {
		t.Fatal("a signal must run the full shutdown, closing the database")
	}
}

func TestServeFailureStillShutsDown(t *testing.T) {
	busy, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer busy.Close()
	app, db := newServeTestApp(t, busy.Addr().(*net.TCPAddr).Port)

	if err := app.serve(context.Background()); err == nil {
		t.Fatal("serving on a taken port must report the error")
	}
	if !db.closed {
		t.Fatal("a serve failure must still shut down services and close the database")
	}
}

// A request still draining must keep a working app context; services must
// see it cancelled before their Cleanup runs.
type ctxWatchService struct {
	Service
	sawCancelled bool
}

func (s *ctxWatchService) Cleanup() error {
	s.sawCancelled = s.AppContext().Err() != nil
	return nil
}

func TestAppContextCancelledAfterDrainBeforeCleanup(t *testing.T) {
	svc := &ctxWatchService{}
	app := NewTestApp(&Components{Services: Services{svc}}, nil)
	if err := app.Core.Resources.AppContext().Err(); err != nil {
		t.Fatalf("the app context must be live while serving: %v", err)
	}
	if app.Core.Resources.ShuttingDown() {
		t.Fatal("not shutting down yet")
	}

	app.Shutdown()

	if !app.Core.Resources.ShuttingDown() {
		t.Fatal("ShuttingDown must report true once Shutdown began")
	}
	if !svc.sawCancelled {
		t.Fatal("services must see the app context cancelled when they clean up")
	}
}

// With shutdown_delay, the app keeps serving after BeginShutdown so load
// balancers see readiness fail before the listener closes.
func TestShutdownDelayServesNotReady(t *testing.T) {
	probe, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	port := probe.Addr().(*net.TCPAddr).Port
	probe.Close()
	app := NewTestApp(&Components{}, router.CollectRoutes(
		router.Get("/readyz", "Health.Ready"),
		router.Get("/healthz", "Health.Live"),
	), WithConfig(&config.Config{ServerConfig: config.ServerConfig{Address: "127.0.0.1", Port: port, ShutdownDelay: 1}}))

	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)
	go func() { done <- app.serve(ctx) }()
	base := fmt.Sprintf("http://127.0.0.1:%d", port)
	waitServing(t, base+"/healthz")

	cancel() // as SIGTERM does
	time.Sleep(200 * time.Millisecond)
	for path, want := range map[string]int{"/readyz": http.StatusServiceUnavailable, "/healthz": http.StatusOK} {
		resp, err := http.Get(base + path)
		if err != nil {
			t.Fatalf("GET %s during the shutdown delay: %v", path, err)
		}
		resp.Body.Close()
		if resp.StatusCode != want {
			t.Errorf("GET %s during the shutdown delay: %d, want %d", path, resp.StatusCode, want)
		}
	}
	if err := <-done; err != nil {
		t.Fatal(err)
	}
}

func waitServing(t *testing.T, url string) {
	t.Helper()
	for range 50 {
		if resp, err := http.Get(url); err == nil {
			resp.Body.Close()
			return
		}
		time.Sleep(20 * time.Millisecond)
	}
	t.Fatalf("%s never answered", url)
}
