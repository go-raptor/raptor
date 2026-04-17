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

	resources      *core.Resources
	testMode       bool
	configOverride *config.Config
}

type RaptorOption func(*Raptor)

func New(components *core.Components, routes router.Routes, opts ...RaptorOption) *Raptor {
	resources := core.NewResources()

	r := &Raptor{
		Router:    router.NewRouter(),
		resources: resources,
	}

	for _, opt := range opts {
		opt(r)
	}

	var cfg *config.Config
	var err error
	if r.testMode {
		cfg, err = config.NewTestConfig(resources.Log)
	} else {
		cfg, err = config.NewConfig(resources.Log)
	}
	if err != nil {
		resources.Log.Error("Failed to load configuration", "error", err)
		panic(err)
	}
	if r.configOverride != nil {
		config.MergeConfig(cfg, r.configOverride)
	}
	resources.SetConfig(cfg)

	r.Core = core.NewCore(resources)
	r.Server = server.NewServer(&r.Core.Resources.Config.ServerConfig, r.Router.Mux)
	r.configure(components)
	r.registerRoutes(routes)

	return r
}

func WithConfig(c *config.Config) RaptorOption {
	return func(r *Raptor) {
		if r.configOverride == nil {
			r.configOverride = c
		} else {
			config.MergeConfig(r.configOverride, c)
		}
	}
}

func WithLogHandler(handler func(*slog.LevelVar) slog.Handler) RaptorOption {
	return func(r *Raptor) {
		r.resources.SetLogHandler(handler(r.resources.LogLevel))
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

	timeout := time.Duration(r.Core.Resources.Config.ServerConfig.ShutdownTimeout) * time.Second
	if timeout <= 0 {
		timeout = time.Duration(config.DefaultServerConfigShutdownTimeout) * time.Second
	}
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
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

func GetService[T any](r *Raptor) *T {
	for _, s := range r.Core.Services {
		if svc, ok := any(s).(*T); ok {
			return svc
		}
	}
	return nil
}

func (r *Raptor) fatal(err error) {
	if err != nil {
		r.Core.Resources.Log.Error("Fatal error", "error", err)
		panic(err)
	}
}
