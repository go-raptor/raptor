package server

import (
	"context"
	"fmt"
	"net/http"

	"github.com/go-raptor/raptor/v4/config"
)

type Server struct {
	server *http.Server
}

func NewServer(config *config.ServerConfig, mux *http.ServeMux) *Server {
	return &Server{
		server: &http.Server{
			Addr:    fmt.Sprintf("%s:%d", config.Address, config.Port),
			Handler: mux,
		},
	}
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
