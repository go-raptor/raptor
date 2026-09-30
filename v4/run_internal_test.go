package raptor

import (
	"context"
	"net"
	"testing"

	"github.com/go-raptor/connectors"
	"github.com/go-raptor/raptor/v4/config"
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
