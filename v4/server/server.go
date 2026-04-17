package server

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/go-raptor/raptor/v4/config"
)

type Server struct {
	server *http.Server
}

func NewServer(cfg *config.ServerConfig, mux *http.ServeMux) *Server {
	return &Server{
		server: &http.Server{
			Addr:              fmt.Sprintf("%s:%d", cfg.Address, cfg.Port),
			Handler:           mux,
			ReadTimeout:       seconds(cfg.ReadTimeout),
			ReadHeaderTimeout: seconds(cfg.ReadHeaderTimeout),
			WriteTimeout:      seconds(cfg.WriteTimeout),
			IdleTimeout:       seconds(cfg.IdleTimeout),
			MaxHeaderBytes:    cfg.MaxHeaderBytes,
		},
	}
}

func seconds(n int) time.Duration {
	if n <= 0 {
		return 0
	}
	return time.Duration(n) * time.Second
}

func (s *Server) Start() error {
	return s.server.ListenAndServe()
}

func (s *Server) Shutdown(ctx context.Context) error {
	return s.server.Shutdown(ctx)
}

func (s *Server) Close() error {
	return s.server.Close()
}

func (s *Server) Address() string {
	return s.server.Addr
}
