package raptor

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"os"
	"os/signal"
	"reflect"
	"strings"
	"sync"

	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
)

type Raptor struct {
	Utils       *Utils
	Server      *echo.Echo
	coordinator *coordinator
	contextPool sync.Pool
	middlewares []MiddlewareInterface
	services    map[string]ServiceInterface
	Routes      Routes
}
type RaptorOption func(*Raptor)

func NewRaptor(opts ...RaptorOption) *Raptor {
	utils := newUtils()
	utils.SetConfig(newConfig(utils.Log))

	raptor := &Raptor{
		Utils:       utils,
		coordinator: newCoordinator(utils),
		contextPool: sync.Pool{
			New: func() interface{} {
				return new(Context)
			},
		},
		middlewares: make([]MiddlewareInterface, 0),
		services:    make(map[string]ServiceInterface),
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

func (r *Raptor) acquireContext(ec echo.Context, controller, action string) *Context {
	ctx := r.contextPool.Get().(*Context)
	ctx.Context = ec
	ctx.Controller = controller
	ctx.Action = action
	return ctx
}

func (r *Raptor) releaseContext(ctx *Context) {
	if ctx == nil {
		return
	}
	ctx.Context = nil
	ctx.Controller = ""
	ctx.Action = ""
	r.contextPool.Put(ctx)
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

	if err := r.registerServices(app); err != nil {
		os.Exit(1)
	}
	if err := r.registerControllers(app); err != nil {
		os.Exit(1)
	}
	if err := r.registerMiddlewares(app); err != nil {
		os.Exit(1)
	}
	r.registerRoutes(app)

	return r
}

func (r *Raptor) registerServices(app *AppInitializer) error {
	for _, service := range app.Services {
		if err := service.InitService(r); err != nil {
			r.Utils.Log.Error("Service initialization failed", "service", reflect.TypeOf(service).Elem().Name(), "error", err)
			return err
		}
		r.services[reflect.TypeOf(service).Elem().Name()] = service
	}

	for _, service := range r.services {
		serviceValue := reflect.ValueOf(service).Elem()
		serviceType := reflect.TypeOf(service).Elem()

		for i := 0; i < serviceValue.NumField(); i++ {
			field := serviceValue.Field(i)
			fieldType := serviceType.Field(i)

			if fieldType.Type.Kind() != reflect.Ptr || fieldType.Type.Elem().Kind() != reflect.Struct {
				continue
			}

			if injectedService, ok := r.services[fieldType.Type.Elem().Name()]; ok {
				field.Set(reflect.ValueOf(injectedService))
				continue
			}

			serviceInterfaceType := reflect.TypeOf((*ServiceInterface)(nil)).Elem()
			if fieldType.Type.Implements(serviceInterfaceType) {
				err := fmt.Errorf("%s requires %s, but the service was not found in services initializer", serviceType.Name(), fieldType.Type.Elem().Name())
				r.Utils.Log.Error("Error while registering service", "service", serviceType.Name(), "error", err)
				return err
			}
		}
	}

	return nil
}

func (r *Raptor) registerMiddlewares(app *AppInitializer) error {
	for _, scopedMiddleware := range app.Middlewares {
		scopedMiddleware.middleware.InitMiddleware(r)
		r.middlewares = append(r.middlewares, scopedMiddleware.middleware)
	}

	for _, middleware := range r.middlewares {
		middlewareValue := reflect.ValueOf(middleware).Elem()
		middlewareType := reflect.TypeOf(middleware).Elem()

		for i := 0; i < middlewareValue.NumField(); i++ {
			field := middlewareValue.Field(i)
			fieldType := middlewareType.Field(i)

			if fieldType.Type.Kind() != reflect.Ptr || fieldType.Type.Elem().Kind() != reflect.Struct {
				continue
			}

			serviceName := fieldType.Type.Elem().Name()
			if injectedService, ok := r.services[serviceName]; ok {
				field.Set(reflect.ValueOf(injectedService))
				continue
			}

			serviceInterfaceType := reflect.TypeOf((*ServiceInterface)(nil)).Elem()
			if fieldType.Type.Implements(serviceInterfaceType) {
				err := fmt.Errorf("%s requires %s, but the service was not found in services initializer", middlewareType.Name(), serviceName)
				r.Utils.Log.Error("Error while registering middleware", "middleware", middlewareType.Name(), "error", err)
				return err
			}
		}
	}

	for _, actions := range r.coordinator.handlers {
		for _, handler := range actions {
			handler.middlewares = make([]uint8, len(r.middlewares))
			for i := range r.middlewares {
				handler.middlewares[i] = uint8(i)
			}
		}
	}

	return nil
}

func (r *Raptor) registerControllers(app *AppInitializer) error {
	for _, controller := range app.Controllers {
		if err := r.coordinator.registerController(controller, r.Utils, r.services); err != nil {
			r.Utils.Log.Error("Error while registering controller", "controller", reflect.TypeOf(controller).Elem().Name(), "error", err)
			return err
		}
	}

	return nil
}

func (r *Raptor) registerRoutes(app *AppInitializer) {
	r.Routes = app.Routes
	for _, route := range r.Routes {
		if _, ok := r.coordinator.handlers[route.Controller][route.Action]; !ok {
			r.Utils.Log.Error(fmt.Sprintf("Action %s not found in controller %s for path %s!", route.Action, route.Controller, route.Path))
			os.Exit(1)
		}
		r.registerRoute(route)
	}
}

func (r *Raptor) registerRoute(route route) {
	routeHandler := r.CreateActionWrapper(route.Controller, route.Action, r.coordinator.handle)
	if route.Method != "*" {
		r.Server.Add(route.Method, route.Path, routeHandler)
		return
	}
	r.Server.Any(route.Path, routeHandler)
}
