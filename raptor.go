package main

import (
	"fmt"
	"net/http"
	"os"
	"os/signal"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/template/html/v2"
)

type Raptor struct {
	server *fiber.App
}

func NewRaptor() *Raptor {
	server := newServer()

	return &Raptor{
		server: server,
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
	fmt.Println("=> Starting Raptor...")
	go func() {
		if err := r.server.Listen("127.0.0.1:7000"); err != nil && err != http.ErrServerClosed {
			panic(err)
		}
	}()
	r.info()
	r.waitForShutdown()
}

func (r *Raptor) info() {
	fmt.Println("=> Raptor is running!")
	fmt.Println("* Listening on http://127.0.0.1:7000")
}

func (r *Raptor) waitForShutdown() {
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, os.Interrupt)
	<-quit
	fmt.Println("=> Shutting down Raptor...")
	if err := r.server.Shutdown(); err != nil {
		fmt.Println("Server Shutdown:", err)
	}
	fmt.Println("=> Raptor exited, bye bye!")
}
