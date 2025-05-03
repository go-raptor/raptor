package server

import (
	"context"
	"fmt"
	"net"
	"net/http"

	"github.com/go-raptor/raptor/v4/config"
	"github.com/go-raptor/raptor/v4/core"
)

type Server struct {
	server  *http.Server
	address string
}

func NewServer(config *config.ServerConfig, mux *http.ServeMux, core *core.Core) *Server {
	address := fmt.Sprintf("%s:%d", config.Address, config.Port)
	return &Server{
		address: address,
		server: &http.Server{
			Addr:    address,
			Handler: mux,
		},
	}
}

func (s *Server) Start() error {
	if !s.checkPort() {
		return fmt.Errorf("port %s is already in use", s.address)
	}
	return s.server.ListenAndServe()
}

func (s *Server) Shutdown(ctx context.Context) error {
	return s.server.Shutdown(ctx)
}

func (s *Server) Close() error {
	return s.server.Close()
}

func (s *Server) checkPort() bool {
	ln, err := net.Listen("tcp", s.address)
	if err == nil {
		ln.Close()
	}
	return err == nil
}

func (s *Server) Address() string {
	return s.address
}
