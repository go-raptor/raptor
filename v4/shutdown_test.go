package raptor_test

import (
	"slices"
	"testing"

	"github.com/go-raptor/connectors"
	"github.com/go-raptor/raptor/v4"
	"github.com/go-raptor/raptor/v4/config"
)

type fakeConnector struct {
	inited bool
	closed bool
}

func (f *fakeConnector) SetConfig(config any)          {}
func (f *fakeConnector) Init() error                   { f.inited = true; return nil }
func (f *fakeConnector) Conn() any                     { return nil }
func (f *fakeConnector) Migrator() connectors.Migrator { return nil }
func (f *fakeConnector) Close() error                  { f.closed = true; return nil }

type LifecycleService struct {
	raptor.Service

	events *[]string
}

func (s *LifecycleService) Cleanup() error {
	*s.events = append(*s.events, "cleanup")
	return nil
}

func (s *LifecycleService) Shutdown() error {
	*s.events = append(*s.events, "shutdown")
	return nil
}

func TestShutdownClosesDatabaseAndRunsHooks(t *testing.T) {
	conn := &fakeConnector{}
	var events []string
	app := raptor.NewTestApp(
		&raptor.Components{
			DatabaseConnector: conn,
			Services:          raptor.Services{&LifecycleService{events: &events}},
		},
		nil,
		raptor.WithConfig(&config.Config{
			DatabaseConfig: config.DatabaseConfig{Host: "localhost", Name: "test"},
		}),
	)
	if !conn.inited {
		t.Fatal("connector should be initialized at startup")
	}

	app.Shutdown()

	if !slices.Equal(events, []string{"cleanup", "shutdown"}) {
		t.Fatalf("service hooks: got %v, want [cleanup shutdown]", events)
	}
	if !conn.closed {
		t.Fatal("database connector implementing io.Closer should be closed during shutdown")
	}
}
