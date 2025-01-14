package raptor

import "github.com/go-raptor/connector"

type AppInitializer struct {
	Routes            Routes
	DatabaseConnector connector.DatabaseConnector
	Middlewares       Middlewares
	Services          Services
	Controllers       Controllers
	Template          Template
}
