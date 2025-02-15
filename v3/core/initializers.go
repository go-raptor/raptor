package core

import (
	"github.com/go-raptor/connector"
	"github.com/go-raptor/raptor/v3/router"
)

type App struct {
	Routes            router.Routes
	DatabaseConnector connector.DatabaseConnector
	Middlewares       Middlewares
	Services          Services
	Controllers       Controllers
}
