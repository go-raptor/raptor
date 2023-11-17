package raptor

import (
	"net"
	"net/http"
	"os"
	"os/signal"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/template/jet/v2"
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

func (r *Raptor) Start() {
	r.Services.Log.Info("====> Starting Raptor <====")
	if r.checkPort("127.0.0.1:3000") {
		go func() {
			if err := r.server.Listen("127.0.0.1:3000"); err != nil && err != http.ErrServerClosed {
				panic(err)
			}
		}()
		r.info()
		r.waitForShutdown()
	} else {
		r.Services.Log.Error("Port 3000 is already in use!")
	}
}

func (r *Raptor) checkPort(addr string) bool {
	ln, err := net.Listen("tcp", addr)
	if err == nil {
		ln.Close()
	}
	return err == nil
}

func (r *Raptor) info() {
	r.Services.Log.Info("Raptor is running! 🎉")
	r.Services.Log.Info("Listening on http://127.0.0.1:3000")
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

func (r *Raptor) Route(method string, path string, handler func(*Context) error) {
	r.server.Get(path, wrapHandler(handler))
}
