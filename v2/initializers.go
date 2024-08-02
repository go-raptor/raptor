package raptor

type AppInitializer struct {
	Routes      Routes
	Database    Database
	Middlewares Middlewares
	Services    Services
	Controllers Controllers
	Template    Template
}
