package core

import (
	"context"
	"fmt"
	"net"
	"net/http"

	"github.com/go-raptor/config"
)

// Server manages an HTTP server instance.
type Server struct {
	server  *http.Server
	address string
}

// NewServer creates a new Server with the given configuration and handler.
func NewServer(config *config.ServerConfig, handler http.Handler) *Server {
	addr := address(config)
	return &Server{
		address: addr,
		server: &http.Server{
			Addr:    addr,
			Handler: handler,
		},
	}
}

// Start begins listening and serving HTTP requests.
// It returns an error if the port is already in use.
func (s *Server) Start() error {
	if !s.checkPort() {
		return fmt.Errorf("port %s is already in use", s.address)
	}
	return s.server.ListenAndServe()
}

// Address returns the server's listening address.
func (s *Server) Address() string {
	return s.address
}

// Shutdown gracefully stops the server with the given context.
func (s *Server) Shutdown(ctx context.Context) error {
	return s.server.Shutdown(ctx)
}

// Close immediately closes the server.
func (s *Server) Close() error {
	return s.server.Close()
}

// checkPort checks if the server's address is available.
func (s *Server) checkPort() bool {
	ln, err := net.Listen("tcp", s.address)
	if err == nil {
		ln.Close()
	}
	return err == nil
}

// address constructs the server address from the configuration.
func address(config *config.ServerConfig) string {
	return config.Address + ":" + fmt.Sprint(config.Port)
}
