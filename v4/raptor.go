package raptor

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/go-raptor/config"
	"github.com/go-raptor/raptor/v4/core"
	"github.com/go-raptor/raptor/v4/router"
	"github.com/go-raptor/raptor/v4/server"
)

type Raptor struct {
	Server *server.Server
	Core   *core.Core
	Router *router.Router
}
type RaptorOption func(*Raptor)

func New(opts ...RaptorOption) *Raptor {
	utils := core.NewUtils()
	config, err := config.NewConfig(utils.Log)
	if err != nil {
		os.Exit(1)
	}
	utils.SetConfig(config)

	core := core.NewCore(utils)
	router, err := router.NewRouter()
	raptor := &Raptor{
		Core:   core,
		Server: server.NewServer(&config.ServerConfig, router.Mux, core),
		Router: router,
	}

	for _, opt := range opts {
		opt(raptor)
	}

	return raptor
}

func WithConfig(c *config.Config) RaptorOption {
	return func(r *Raptor) {
		if c != nil {
			config.MergeConfig(r.Core.Utils.Config, c)
		}
	}
}

func (r *Raptor) Run() {
	go func() {
		if err := r.Server.Start(); err != nil && err != http.ErrServerClosed {
			panic(err)
		}
	}()
	r.info()
	r.waitForShutdown()
}

func (r *Raptor) info() {
	logo := `██████╗  █████╗ ██████╗ ████████╗ ██████╗ ██████╗ 
██╔══██╗██╔══██╗██╔══██╗╚══██╔══╝██╔═══██╗██╔══██╗
██████╔╝███████║██████╔╝   ██║   ██║   ██║██████╔╝
██╔══██╗██╔══██║██╔═══╝    ██║   ██║   ██║██╔══██╗
██║  ██║██║  ██║██║        ██║   ╚██████╔╝██║  ██║
╚═╝  ╚═╝╚═╝  ╚═╝╚═╝        ╚═╝    ╚═════╝ ╚═╝  ╚═╝`
	content := []string{
		"Raptor is running! 🦖💨",
		logo,
		fmt.Sprintf("🟢 Raptor %s running on %s", Version, r.Server.Address()),
	}

	r.Core.Utils.Log.Info(strings.Join(content, "\n"))
}

func (r *Raptor) waitForShutdown() {
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, os.Interrupt, syscall.SIGTERM)

	<-quit
	r.Core.Utils.Log.Warn("Shutting down Raptor...")

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := r.Server.Shutdown(ctx); err != nil {
		r.Core.Utils.Log.Error("Server shutdown", "error", err)
		if err := r.Server.Close(); err != nil {
			r.Core.Utils.Log.Error("Server force close", "error", err)
		}
	}

	r.Core.Utils.Log.Warn("Raptor exited, bye bye!")
}

func (r *Raptor) Configure() error {
	if err := r.Router.RegisterRoutes(r.Router.Routes, r.Core); err != nil {
		r.Core.Utils.Log.Error("Error while registering routes", "error", err)
		os.Exit(1)
	}

	return nil
}

func (r *Raptor) RegisterRoutes(routes router.Routes) {
	if err := r.Router.RegisterRoutes(routes, r.Core); err != nil {
		r.Core.Utils.Log.Error("Error while registering routes", "error", err)
		os.Exit(1)
	}
}
