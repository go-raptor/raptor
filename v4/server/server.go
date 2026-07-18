package server

import (
	"context"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"time"

	"github.com/go-raptor/raptor/v4/config"
)

type Server struct {
	server   *http.Server
	listener net.Listener
}

func NewServer(cfg *config.ServerConfig, mux *http.ServeMux, log *slog.Logger) *Server {
	return &Server{
		server: &http.Server{
			Addr:              fmt.Sprintf("%s:%d", cfg.Address, cfg.Port),
			Handler:           mux,
			ReadTimeout:       seconds(cfg.ReadTimeout),
			ReadHeaderTimeout: seconds(cfg.ReadHeaderTimeout),
			WriteTimeout:      seconds(cfg.WriteTimeout),
			IdleTimeout:       seconds(cfg.IdleTimeout),
			MaxHeaderBytes:    cfg.MaxHeaderBytes,
			ErrorLog:          slog.NewLogLogger(log.Handler(), slog.LevelWarn),
		},
	}
}

func seconds(n int) time.Duration {
	if n <= 0 {
		return 0
	}
	return time.Duration(n) * time.Second
}

// Listen binds the configured address without serving yet, so bind
// errors surface synchronously before the app reports itself running.
func (s *Server) Listen() error {
	listener, err := net.Listen("tcp", s.server.Addr)
	if err != nil {
		return err
	}
	s.listener = listener
	return nil
}

// Serve accepts connections on the bound listener, binding first if
// Listen has not been called.
func (s *Server) Serve() error {
	if s.listener == nil {
		if err := s.Listen(); err != nil {
			return err
		}
	}
	return s.server.Serve(s.listener)
}

func (s *Server) Start() error {
	return s.Serve()
}

func (s *Server) Shutdown(ctx context.Context) error {
	return s.server.Shutdown(ctx)
}

func (s *Server) Close() error {
	return s.server.Close()
}

// Address returns the bound address once listening (reporting the real
// port when the config asked for :0), or the configured address before.
func (s *Server) Address() string {
	if s.listener != nil {
		return s.listener.Addr().String()
	}
	return s.server.Addr
}
