package raptor

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/go-raptor/raptor/v4/config"
	"github.com/go-raptor/raptor/v4/core"
	"github.com/go-raptor/raptor/v4/router"
	"github.com/go-raptor/raptor/v4/server"
)

type Raptor struct {
	Core   *core.Core
	Server *server.Server
	Router *router.Router
}

type RaptorOption func(*Raptor)

func New(components *core.Components, routes router.Routes, opts ...RaptorOption) *Raptor {
	resources := core.NewResources()
	cfg, err := config.NewConfig(resources.Log)
	if err != nil {
		os.Exit(1)
	}
	resources.SetConfig(cfg)

	r := &Raptor{
		Core:   core.NewCore(resources),
		Router: router.NewRouter(),
	}

	for _, opt := range opts {
		opt(r)
	}

	r.Server = server.NewServer(&r.Core.Resources.Config.ServerConfig, r.Router.Mux)
	r.configure(components)
	r.registerRoutes(routes)

	return r
}

func WithConfig(c *config.Config) RaptorOption {
	return func(r *Raptor) {
		if c != nil {
			config.MergeConfig(r.Core.Resources.Config, c)
		}
	}
}

func WithLogHandler(handler func(*slog.LevelVar) slog.Handler) RaptorOption {
	return func(r *Raptor) {
		r.Core.Resources.SetLogHandler(handler(r.Core.Resources.LogLevel))
	}
}

func (r *Raptor) Run() {
	go func() {
		if err := r.Server.Start(); err != nil && err != http.ErrServerClosed {
			r.Core.Resources.Log.Error("Error while starting Raptor", "error", err)
			os.Exit(1)
		}
	}()
	r.Core.Resources.Log.Info(fmt.Sprintf("🟢 Raptor %s is running on %s! 🦖💨", Version, r.Server.Address()))
	r.waitForShutdown()
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

func (r *Raptor) configure(components *core.Components) {
	r.initDatabase(components)
	r.fatal(r.Core.RegisterServices(components))
	r.fatal(r.Core.RegisterControllers(components))
	r.fatal(r.Core.RegisterMiddlewares(components))
}

func (r *Raptor) initDatabase(components *core.Components) {
	if components.DatabaseConnector == nil {
		return
	}
	if !r.Core.Resources.Config.DatabaseConfig.IsConfigured() {
		r.fatal(fmt.Errorf("database connector registered but no database configuration found — configure via .raptor.yaml or DATABASE_* environment variables"))
	}
	r.Core.Resources.Database = components.DatabaseConnector
	r.Core.Resources.Database.SetConfig(r.Core.Resources.Config.DatabaseConfig)
	r.fatal(r.Core.Resources.Database.Init())
}

func (r *Raptor) registerRoutes(routes router.Routes) {
	r.fatal(r.Router.RegisterRoutes(routes, r.Core))
}

func (r *Raptor) fatal(err error) {
	if err != nil {
		r.Core.Resources.Log.Error("Fatal error", "error", err)
		os.Exit(1)
	}
}
