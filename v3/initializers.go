package raptor

type AppInitializer struct {
	Routes            Routes
	DatabaseConnector DatabaseConnector
	Middlewares       Middlewares
	Services          Services
	Controllers       Controllers
	Template          Template
}
