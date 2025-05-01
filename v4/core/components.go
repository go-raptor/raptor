package core

import (
	"github.com/go-raptor/connectors"
)

type Components struct {
	DatabaseConnector connectors.DatabaseConnector
	Controllers       Controllers
	Services          Services
	//Middlewares       Middlewares
}
