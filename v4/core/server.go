package core

import (
	"context"
	"fmt"
	"net"
	"net/http"

	"github.com/go-raptor/config"
)

type Server struct {
	server  *http.Server
	address string
}

func NewServer(config *config.ServerConfig, mux *http.ServeMux, core *Core) *Server {
	addr := address(config)
	return &Server{
		address: addr,
		server: &http.Server{
			Addr:    addr,
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

func (s *Server) Address() string {
	return s.address
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

func address(config *config.ServerConfig) string {
	return config.Address + ":" + fmt.Sprint(config.Port)
}
