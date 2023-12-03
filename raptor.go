package raptor

import (
	"fmt"
	"net"
	"net/http"
	"os"
	"os/signal"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/template/jet/v2"
)

type Raptor struct {
	Services    *Services
	config      Config
	server      *fiber.App
	controllers Controllers
	routes      Routes
}

func NewRaptorMVC(userConfig ...Config) *Raptor {
	server := newServerMVC()

	raptor := &Raptor{
		config:   config(userConfig...),
		server:   server,
		Services: NewServices(),
	}

	return raptor
}

func NewRaptorAPI(userConfig ...Config) *Raptor {
	server := newServerAPI()

	raptor := &Raptor{
		config:   config(userConfig...),
		server:   server,
		Services: NewServices(),
	}

	return raptor
}

func (r *Raptor) Listen() {
	r.Services.Log.Info("====> Starting Raptor <====")
	if r.checkPort() {
		go func() {
			if err := r.server.Listen(r.address()); err != nil && err != http.ErrServerClosed {
				panic(err)
			}
		}()
		r.info()
		r.waitForShutdown()
	} else {
		r.Services.Log.Error(fmt.Sprintf("Unable to bind on address %s, already in use!", r.address()))
	}
}

func (r *Raptor) address() string {
	return r.config.Address + ":" + fmt.Sprint(r.config.Port)
}

func (r *Raptor) checkPort() bool {
	ln, err := net.Listen("tcp", r.address())
	if err == nil {
		ln.Close()
	}
	return err == nil
}

func newServerMVC() *fiber.App {
	engine := jet.New("app/views", ".html.jet")

	// TODO: add this to the config
	engine.Reload(true)

	server := fiber.New(fiber.Config{
		DisableStartupMessage: true,
		Views:                 engine,
		ViewsLayout:           "layouts/main",
	})
	server.Static("/public", "./public")

	return server
}

func newServerAPI() *fiber.App {
	server := fiber.New(fiber.Config{
		DisableStartupMessage: true,
	})

	return server
}

func (r *Raptor) info() {
	r.Services.Log.Info("Raptor is running! 🎉")
	r.Services.Log.Info(fmt.Sprintf("Listening on http://%s", r.address()))
}

func (r *Raptor) waitForShutdown() {
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, os.Interrupt)
	<-quit
	r.Services.Log.Warn("Shutting down Raptor...")
	if err := r.server.Shutdown(); err != nil {
		r.Services.Log.Error("Server Shutdown:", err)
	}
	r.Services.Log.Warn("Raptor exited, bye bye!")
}

func (r *Raptor) Controllers(c Controllers) {
	r.controllers = c
	for _, controller := range r.controllers {
		controller.SetServices(r)
	}
}

func (r *Raptor) Routes(routes Routes) {
	r.routes = routes
	for _, route := range r.routes {
		if _, ok := r.controllers[route.Controller]; !ok {
			r.Services.Log.Error(fmt.Sprintf("Controller %s not found for path %s!", route.Controller, route.Path))
			os.Exit(1)
		}
		if _, ok := r.controllers[route.Controller].Actions[route.Action]; !ok {
			r.Services.Log.Error(fmt.Sprintf("Action %s not found in controller %s for path %s!", route.Action, route.Controller, route.Path))
			os.Exit(1)
		}
		r.route(route.Method, route.Path, route.Controller, route.Action)
	}
}

func (r *Raptor) route(method, path, controller, action string) {
	r.server.Add(method, path, wrapHandler(action, r.controllers[controller].Action))
}
