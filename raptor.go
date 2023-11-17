package raptor

import (
	"net/http"
	"os"
	"os/signal"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/template/html/v2"
)

type Raptor struct {
	server   *fiber.App
	Services *Services
}

func NewRaptor() *Raptor {
	server := newServer()

	return &Raptor{
		server:   server,
		Services: NewServices(),
	}
}

func newServer() *fiber.App {
	engine := html.New("app/views", ".html")

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

func (r *Raptor) Start() {
	r.Services.Log.Info("====> Starting Raptor <====")
	go func() {
		if err := r.server.Listen("127.0.0.1:7000"); err != nil && err != http.ErrServerClosed {
			panic(err)
		}
	}()
	r.info()
	r.waitForShutdown()
}

func (r *Raptor) info() {
	r.Services.Log.Info("Raptor is running! 🎉")
	r.Services.Log.Info("Listening on http://127.0.0.1:7000")
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
