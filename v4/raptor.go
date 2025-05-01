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
	"github.com/go-raptor/raptor/v4/components"
	"github.com/go-raptor/raptor/v4/core"
	"github.com/go-raptor/raptor/v4/router"
	"github.com/go-raptor/raptor/v4/server"
)

type Raptor struct {
	Core      *core.Core
	Server    *server.Server
	Router    *router.Router
	Resources *components.Resources
}
type RaptorOption func(*Raptor)

func New(opts ...RaptorOption) *Raptor {
	resources := components.NewResources()
	config, err := config.NewConfig(resources.Log)
	if err != nil {
		os.Exit(1)
	}
	resources.SetConfig(config)

	core := core.NewCore(resources)
	router, err := router.NewRouter()
	raptor := &Raptor{
		Core:      core,
		Server:    server.NewServer(&config.ServerConfig, router.Mux, core),
		Router:    router,
		Resources: resources,
	}

	for _, opt := range opts {
		opt(raptor)
	}

	return raptor
}

func WithConfig(c *config.Config) RaptorOption {
	return func(r *Raptor) {
		if c != nil {
			config.MergeConfig(r.Core.Resources.Config, c)
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

	r.Core.Resources.Log.Info(strings.Join(content, "\n"))
}

func (r *Raptor) waitForShutdown() {
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, os.Interrupt, syscall.SIGTERM)

	<-quit
	r.Core.Resources.Log.Warn("Shutting down Raptor...")

	if err := r.Core.ShutdownServices(); err != nil {
		r.Core.Resources.Log.Error("Error shutting down services", "error", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := r.Server.Shutdown(ctx); err != nil {
		r.Core.Resources.Log.Error("Server shutdown", "error", err)
		if err := r.Server.Close(); err != nil {
			r.Core.Resources.Log.Error("Server force close", "error", err)
		}
	}

	r.Core.Resources.Log.Warn("Raptor exited, bye bye!")
}

func (r *Raptor) Configure(components *components.Components) error {
	if components.DatabaseConnector != nil {
		r.Resources.DB = components.DatabaseConnector
		if err := r.Resources.DB.Init(); err != nil {
			r.Resources.Log.Error("Database connector initalization failed", "error", err)
			os.Exit(1)
		}
	}

	if err := r.Core.RegisterServices(components); err != nil {
		os.Exit(1)
	}

	if err := r.Core.RegisterControllers(components); err != nil {
		os.Exit(1)
	}

	return nil
}

func (r *Raptor) RegisterRoutes(routes router.Routes) {
	if err := r.Router.RegisterRoutes(routes, r.Core); err != nil {
		r.Core.Resources.Log.Error("Error while registering routes", "error", err)
		os.Exit(1)
	}
}
