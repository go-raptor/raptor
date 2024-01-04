package raptor

import (
	"fmt"
	"net"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
	"github.com/gofiber/template/html/v2"
)

type Raptor struct {
	Utils       *Utils
	config      *Config
	server      *fiber.App
	controllers Controllers
	routes      Routes
}

func NewRaptor() *Raptor {
	utils := newUtils()
	config := newConfig(utils)

	raptor := &Raptor{
		config: config,
		server: newServer(config),
		Utils:  utils,
	}

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
	return r.config.Server.Address + ":" + fmt.Sprint(r.config.Server.Port)
}

func (r *Raptor) checkPort() bool {
	ln, err := net.Listen("tcp", r.address())
	if err == nil {
		ln.Close()
	}
	return err == nil
}

func newServer(config *Config) *fiber.App {
	var server *fiber.App
	if config.Templating.Enabled {
		server = newServerMVC(config)
	} else {
		server = newServerAPI(config)
	}

	server.Use(cors.New(cors.Config{
		AllowOrigins:     strings.Join(config.CORS.Origins, ", "),
		AllowCredentials: config.CORS.Credentials,
	}))

	if config.Templating.Static {
		server.Static(config.Templating.StaticPrefix, config.Templating.StaticRoot)
	}

	return server
}

func newServerMVC(c *Config) *fiber.App {
	engine := html.New("./app/views", ".html")

	engine.Reload(c.Templating.Reload)

	server := fiber.New(fiber.Config{
		DisableStartupMessage: true,
		Views:                 engine,
		ViewsLayout:           "layouts/main",
	})

	return server
}

func newServerAPI(c *Config) *fiber.App {
	server := fiber.New(fiber.Config{
		DisableStartupMessage: true,
	})

	return server
}

func (r *Raptor) info() {
	if r.config.General.Development {
		r.Utils.Log.Info("Raptor is running (development)! 🎉")
	} else {
		r.Utils.Log.Info("Raptor is running (production)! 🎉")
	}
	r.Utils.Log.Info(fmt.Sprintf("Listening on http://%s", r.address()))
}

func (r *Raptor) waitForShutdown() {
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, os.Interrupt)
	<-quit
	r.Utils.Log.Warn("Shutting down Raptor...")
	if err := r.server.ShutdownWithTimeout(time.Duration(r.config.Server.ShutdownTimeout) * time.Second); err != nil {
		r.Utils.Log.Error("Server Shutdown:", err)
	}
	r.Utils.Log.Warn("Raptor exited, bye bye!")
}

func (r *Raptor) Middlewares(middlewares Middlewares) {
	for _, middleware := range middlewares {
		middleware.SetUtils(r.Utils)
		r.server.Use(wrapHandler(middleware.New))
	}
}

func (r *Raptor) Services(services Services) {
	for _, service := range services {
		service.SetUtils(r.Utils)
	}
}

func (r *Raptor) Controllers(c Controllers) {
	r.controllers = c
	for _, controller := range r.controllers {
		controller.SetUtils(r)
	}
}

func (r *Raptor) Routes(routes Routes) {
	r.routes = routes
	for _, route := range r.routes {
		if _, ok := r.controllers[route.Controller]; !ok {
			r.Utils.Log.Error(fmt.Sprintf("Controller %s not found for path %s!", route.Controller, route.Path))
			os.Exit(1)
		}
		if _, ok := r.controllers[route.Controller].Actions[route.Action]; !ok {
			r.Utils.Log.Error(fmt.Sprintf("Action %s not found in controller %s for path %s!", route.Action, route.Controller, route.Path))
			os.Exit(1)
		}
		r.route(route)
	}
}

func (r *Raptor) route(route route) {
	r.server.Add(route.Method, route.Path, wrapActionHandler(route.Controller, route.Action, r.controllers[route.Controller].action))
}
