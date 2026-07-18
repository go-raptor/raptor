package server

import (
	"io"
	"log/slog"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/go-raptor/raptor/v4/config"
)

func testServer(port int) *Server {
	cfg := config.NewConfigDefaults().ServerConfig
	cfg.Port = port
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	return NewServer(&cfg, http.NewServeMux(), logger)
}

func TestNewServerConfiguresErrorLogAndTimeouts(t *testing.T) {
	s := testServer(0)

	if s.server.ErrorLog == nil {
		t.Fatal("http.Server.ErrorLog should be wired to slog so server errors are not lost")
	}
	if s.server.ReadHeaderTimeout != 10*time.Second {
		t.Fatalf("ReadHeaderTimeout: got %v", s.server.ReadHeaderTimeout)
	}
	if s.server.IdleTimeout != 120*time.Second {
		t.Fatalf("IdleTimeout: got %v", s.server.IdleTimeout)
	}
	if s.server.MaxHeaderBytes != 1<<20 {
		t.Fatalf("MaxHeaderBytes: got %d", s.server.MaxHeaderBytes)
	}
}

func TestListenReportsRealAddressAndServes(t *testing.T) {
	s := testServer(0)

	if err := s.Listen(); err != nil {
		t.Fatalf("Listen: %v", err)
	}
	defer s.Close()

	addr := s.Address()
	if strings.HasSuffix(addr, ":0") {
		t.Fatalf("Address should report the actually bound port after Listen: %s", addr)
	}

	go s.Serve() //nolint:errcheck // returns on Close

	resp, err := http.Get("http://" + addr + "/")
	if err != nil {
		t.Fatalf("server not reachable on %s: %v", addr, err)
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusNotFound {
		t.Fatalf("empty mux should 404: got %d", resp.StatusCode)
	}
}
