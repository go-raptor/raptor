package raptor

import (
	"fmt"
	"net"
	"net/http"
	"os"
	"os/signal"
	"reflect"
	"time"

	"github.com/gofiber/fiber/v3"
	"github.com/gofiber/fiber/v3/middleware/cors"
	"github.com/gofiber/fiber/v3/middleware/static"
)

type Raptor struct {
	Utils       *Utils
	Server      *fiber.App
	coordinator *coordinator
	middlewares map[string]MiddlewareInterface
	services    map[string]ServiceInterface
	Routes      Routes
}

func NewRaptor() *Raptor {
	utils := newUtils()
	utils.SetConfig(newConfig(utils.Log))

	raptor := &Raptor{
		Utils:       utils,
		coordinator: newCoordinator(utils),
		middlewares: make(map[string]MiddlewareInterface),
		services:    make(map[string]ServiceInterface),
	}

	return raptor
}

func (r *Raptor) Listen() {
	r.Utils.Log.Info("====> Starting Raptor <====")
	if r.checkPort() {
		go func() {
			if err := r.Server.Listen(r.address(), fiber.ListenConfig{DisableStartupMessage: true}); err != nil && err != http.ErrServerClosed {
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

func newServer(config *Config, app *AppInitializer) *fiber.App {
	var server *fiber.App
	if config.TemplatingConfig.Enabled {
		server = newServerMVC(config, app)
	} else {
		server = newServerAPI(config, app)
	}

	server.Use(cors.New(cors.Config{
		AllowOrigins:     config.CORSConfig.Origins,
		AllowCredentials: config.CORSConfig.Credentials,
	}))

	if config.StaticConfig.Enabled {
		server.Get(config.StaticConfig.Prefix, static.New(config.StaticConfig.Root))
	}

	return server
}

func newServerMVC(config *Config, app *AppInitializer) *fiber.App {
	engine := app.Template.Engine

	server := fiber.New(fiber.Config{
		//DisableStartupMessage: true,
		Views:       engine,
		ViewsLayout: app.Template.Layout,
		ProxyHeader: config.ServerConfig.ProxyHeader,
	})

	return server
}

func newServerAPI(config *Config, _ *AppInitializer) *fiber.App {
	server := fiber.New(fiber.Config{
		//DisableStartupMessage: true,
		ProxyHeader: config.ServerConfig.ProxyHeader,
	})

	return server
}

func (r *Raptor) info() {
	if r.Utils.Config.GeneralConfig.Development {
		r.Utils.Log.Info(fmt.Sprintf("Raptor %v is running (development)! 🎉", Version))
	} else {
		r.Utils.Log.Info(fmt.Sprintf("Raptor %v is running (production)! 🎉", Version))
	}
	r.Utils.Log.Info(fmt.Sprintf("Listening on http://%s", r.address()))
}

func (r *Raptor) waitForShutdown() {
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, os.Interrupt)
	<-quit
	r.Utils.Log.Warn("Shutting down Raptor...")
	if err := r.Server.ShutdownWithTimeout(time.Duration(r.Utils.Config.ServerConfig.ShutdownTimeout) * time.Second); err != nil {
		r.Utils.Log.Error("Server Shutdown:", err)
	}
	r.Utils.Log.Warn("Raptor exited, bye bye!")
}

func (r *Raptor) Init(app *AppInitializer) *Raptor {
	r.Server = newServer(r.Utils.Config, app)
	if r.Utils.Config.DatabaseConfig.Type != "none" {
		r.initDB(newDB(app.Database))
	}
	r.registerServices(app)
	r.registerMiddlewares(app)
	r.registerControllers(app)
	r.registerRoutes(app)

	for _, service := range r.services {
		service.InitService(r)
	}
	for _, middleware := range r.middlewares {
		middleware.InitMiddleware(r)
	}

	return r
}

func (r *Raptor) initDB(db *DB) {
	if db != nil {
		gormDB, err := db.Connector.Connect(r.Utils.Config.DatabaseConfig)
		if err != nil {
			r.Utils.Log.Error("Database connection failed:", err)
			os.Exit(1)
		}
		db.DB = gormDB
		err = db.migrate()
		if err != nil {
			r.Utils.Log.Error("Database migration failed:", err)
			os.Exit(1)
		}
		r.Utils.SetDB(db)
	}
}

func (r *Raptor) registerServices(app *AppInitializer) {
	for _, service := range app.Services {
		r.services[reflect.TypeOf(service).Elem().Name()] = service
	}

	for _, service := range r.services {
		for i := 0; i < reflect.ValueOf(service).Elem().NumField(); i++ {
			field := reflect.ValueOf(service).Elem().Field(i)
			fieldType := reflect.TypeOf(service).Elem().Field(i)
			if fieldType.Type.Kind() == reflect.Ptr && fieldType.Type.Elem().Kind() == reflect.Struct {
				if service, ok := r.services[fieldType.Type.Elem().Name()]; ok {
					field.Set(reflect.ValueOf(service))
				}
			}
		}
	}
}

func (r *Raptor) registerMiddlewares(app *AppInitializer) {
	for _, middleware := range app.Middlewares {
		r.middlewares[reflect.TypeOf(middleware).Elem().Name()] = middleware
		r.Server.Use(wrapHandler(middleware.New))

		for i := 0; i < reflect.ValueOf(middleware).Elem().NumField(); i++ {
			field := reflect.ValueOf(middleware).Elem().Field(i)
			fieldType := reflect.TypeOf(middleware).Elem().Field(i)
			if fieldType.Type.Kind() == reflect.Ptr && fieldType.Type.Elem().Kind() == reflect.Struct {
				if service, ok := r.services[fieldType.Type.Elem().Name()]; ok {
					field.Set(reflect.ValueOf(service))
				}
			}
		}
	}
}

func (r *Raptor) registerControllers(app *AppInitializer) {
	for _, controller := range app.Controllers {
		r.coordinator.registerController(controller, r.Utils, r.services)
	}
}

func (r *Raptor) registerRoutes(app *AppInitializer) {
	r.Routes = app.Routes
	for _, route := range r.Routes {
		if _, ok := r.coordinator.actions[route.Controller][route.Action]; !ok {
			r.Utils.Log.Error(fmt.Sprintf("Action %s not found in controller %s for path %s!", route.Action, route.Controller, route.Path))
			os.Exit(1)
		}
		r.registerRoute(route)
	}
}

func (r *Raptor) registerRoute(route route) {
	r.Server.Add([]string{route.Method}, route.Path, wrapActionHandler(route.Controller, route.Action, r.coordinator.action))
}
