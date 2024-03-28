package raptor

import (
	"fmt"
	"net"
	"net/http"
	"os"
	"os/signal"
	"reflect"
	"strings"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
)

type Raptor struct {
	Utils       *Utils
	config      *Config
	app         *AppInitializer
	server      *fiber.App
	coordinator *coordinator
	svcs        map[string]ServiceInterface
	routes      Routes
}

func NewRaptor(app *AppInitializer, routes Routes) *Raptor {
	utils := newUtils()
	config := newConfig(utils.Log)

	raptor := &Raptor{
		Utils:       utils,
		config:      config,
		app:         app,
		server:      newServer(config, app),
		coordinator: newCoordinator(utils),
		svcs:        make(map[string]ServiceInterface),
		routes:      routes,
	}

	raptor.init()
	raptor.registerRoutes()

	return raptor
}

func (r *Raptor) Listen() {
	r.Utils.Log.Info("====> Starting Raptor <====")
	if r.checkPort() {
		go func() {
			if err := r.server.Listen(r.address()); err != nil && err != http.ErrServerClosed {
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
	return r.config.ServerConfig.Address + ":" + fmt.Sprint(r.config.ServerConfig.Port)
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
		server = newServerMVC(app)
	} else {
		server = newServerAPI()
	}

	server.Use(cors.New(cors.Config{
		AllowOrigins:     strings.Join(config.CORSConfig.Origins, ", "),
		AllowCredentials: config.CORSConfig.Credentials,
	}))

	if config.StaticConfig.Enabled {
		server.Static(config.StaticConfig.Prefix, config.StaticConfig.Root)
	}

	return server
}

func newServerMVC(app *AppInitializer) *fiber.App {
	engine := app.Template.Engine

	server := fiber.New(fiber.Config{
		DisableStartupMessage: true,
		Views:                 engine,
		ViewsLayout:           app.Template.Layout,
	})

	return server
}

func newServerAPI() *fiber.App {
	server := fiber.New(fiber.Config{
		DisableStartupMessage: true,
	})

	return server
}

func (r *Raptor) info() {
	if r.config.GeneralConfig.Development {
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
	if err := r.server.ShutdownWithTimeout(time.Duration(r.config.ServerConfig.ShutdownTimeout) * time.Second); err != nil {
		r.Utils.Log.Error("Server Shutdown:", err)
	}
	r.Utils.Log.Warn("Raptor exited, bye bye!")
}

func (r *Raptor) init() {
	r.Utils.SetConfig(r.config)
	if r.config.DatabaseConfig.Type != "none" {
		r.initDB(newDB(r.app.Database))
	}
	r.registerMiddlewares()
	r.registerServices()
	r.registerControllers()
}

func (r *Raptor) initDB(db *DB) {
	if db != nil {
		gormDB, err := db.Connector.Connect(r.config.DatabaseConfig)
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

func (r *Raptor) registerMiddlewares() {
	for _, middleware := range r.app.Middlewares {
		r.server.Use(wrapHandler(middleware.New))
		middleware.Init(r.Utils)
	}
}

func (r *Raptor) registerServices() {
	for _, service := range r.app.Services {
		service.Init(r.Utils, r.svcs)
		r.svcs[reflect.TypeOf(service).Elem().Name()] = service
	}

	for _, service := range r.svcs {
		for i := 0; i < reflect.ValueOf(service).Elem().NumField(); i++ {
			field := reflect.ValueOf(service).Elem().Field(i)
			fieldType := reflect.TypeOf(service).Elem().Field(i)
			if fieldType.Type.Kind() == reflect.Ptr && fieldType.Type.Elem().Kind() == reflect.Struct {
				if service, ok := r.svcs[fieldType.Type.Elem().Name()]; ok {
					field.Set(reflect.ValueOf(service))
				}
			}
		}
	}
}

func (r *Raptor) registerControllers() {
	for _, controller := range r.app.Controllers {
		r.coordinator.registerController(controller, r.Utils, r.svcs)
	}
}

func (r *Raptor) registerRoutes() {
	for _, route := range r.routes {
		if _, ok := r.coordinator.actions[route.Controller][route.Action]; !ok {
			r.Utils.Log.Error(fmt.Sprintf("Action %s not found in controller %s for path %s!", route.Action, route.Controller, route.Path))
			os.Exit(1)
		}
		r.registerRoute(route)
	}
}

func (r *Raptor) registerRoute(route route) {
	r.server.Add(route.Method, route.Path, wrapActionHandler(route.Controller, route.Action, r.coordinator.action))
}
