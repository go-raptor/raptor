package raptor

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"os"
	"os/signal"
	"strings"

	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
)

type Raptor struct {
	Utils       *Utils
	Server      *echo.Echo
	coordinator *coordinator
	Routes      Routes
}
type RaptorOption func(*Raptor)

func NewRaptor(opts ...RaptorOption) *Raptor {
	utils := newUtils()
	utils.SetConfig(newConfig(utils.Log))

	raptor := &Raptor{
		Utils:       utils,
		coordinator: newCoordinator(utils),
	}

	for _, opt := range opts {
		opt(raptor)
	}

	raptor.Utils.SetLogLevel(utils.Config.GeneralConfig.LogLevel)
	return raptor
}

func (r *Raptor) Listen() {
	if r.checkPort() {
		go func() {
			if err := r.Server.Start(r.address()); err != nil && err != http.ErrServerClosed {
				panic(err)
			}
		}()
		r.info()
		r.waitForShutdown()
	} else {
		r.Utils.Log.Error(fmt.Sprintf("Unable to bind on address %s, already in use!", r.address()))
	}
}

func (r *Raptor) address() string {
	return r.Utils.Config.ServerConfig.Address + ":" + fmt.Sprint(r.Utils.Config.ServerConfig.Port)
}

func (r *Raptor) checkPort() bool {
	ln, err := net.Listen("tcp", r.address())
	if err == nil {
		ln.Close()
	}
	return err == nil
}

func newServer(config *Config, _ *AppInitializer) *echo.Echo {
	server := echo.New()

	switch strings.ToLower(config.ServerConfig.IPExtractor) {
	case "x-forwarded-for":
		server.IPExtractor = echo.ExtractIPFromXFFHeader()
	case "x-real-ip":
		server.IPExtractor = echo.ExtractIPFromRealIPHeader()
	default:
		server.IPExtractor = echo.ExtractIPDirect()
	}

	server.HideBanner = true
	server.HidePort = true

	server.Use(middleware.CORSWithConfig(middleware.CORSConfig{
		AllowOrigins:     config.CORSConfig.AllowOrigins,
		AllowHeaders:     config.CORSConfig.AllowHeaders,
		AllowMethods:     config.CORSConfig.AllowMethods,
		AllowCredentials: config.CORSConfig.AllowCredentials,
		MaxAge:           config.CORSConfig.MaxAge,
	}))

	if config.StaticConfig.Enabled {
		if config.StaticConfig.HTML5 {
			server.Use(middleware.StaticWithConfig(middleware.StaticConfig{
				Root:   config.StaticConfig.Root,
				Index:  config.StaticConfig.Index,
				Browse: config.StaticConfig.Browse,
				HTML5:  config.StaticConfig.HTML5,
			}))
		} else {
			server.Static(config.StaticConfig.Prefix, config.StaticConfig.Root)
		}
	}

	return server
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
		fmt.Sprintf("🟢 Raptor %s started on %s", Version, r.address()),
	}

	r.Utils.Log.Info(strings.Join(content, "\n"))
}

func (r *Raptor) waitForShutdown() {
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, os.Interrupt)
	<-quit
	r.Utils.Log.Warn("Shutting down Raptor...")
	if err := r.Server.Shutdown(context.Background()); err != nil {
		r.Utils.Log.Error("Server Shutdown", "error", err)
	}
	r.Utils.Log.Warn("Raptor exited, bye bye!")
}

func (r *Raptor) Init(app *AppInitializer) *Raptor {
	r.Server = newServer(r.Utils.Config, app)
	if app.DatabaseConnector != nil {
		r.Utils.DB = app.DatabaseConnector
		if err := r.Utils.DB.Init(); err != nil {
			r.Utils.Log.Error("Database initalization failed", "error", err.Error())
			os.Exit(1)
		}
	}

	if err := r.coordinator.registerServices(app); err != nil {
		os.Exit(1)
	}
	if err := r.coordinator.registerControllers(app); err != nil {
		os.Exit(1)
	}
	if err := r.coordinator.registerMiddlewares(app); err != nil {
		os.Exit(1)
	}
	r.registerRoutes(app)

	return r
}

func (r *Raptor) registerRoutes(app *AppInitializer) {
	r.Routes = app.Routes
	for _, route := range r.Routes {
		if !r.coordinator.hasControllerAction(route.Controller, route.Action) {
			r.Utils.Log.Error(fmt.Sprintf("Action %s not found in controller %s for path %s!", route.Action, route.Controller, route.Path))
			os.Exit(1)
		}
		r.registerRoute(route)
	}
}

func (r *Raptor) registerRoute(route route) {
	routeHandler := r.coordinator.CreateActionWrapper(route.Controller, route.Action, r.coordinator.handle)
	if route.Method != "*" {
		r.Server.Add(route.Method, route.Path, routeHandler)
		return
	}
	r.Server.Any(route.Path, routeHandler)
}
