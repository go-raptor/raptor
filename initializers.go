package raptor

type AppInitializer struct {
	Database    Database
	Middlewares Middlewares
	Services    Services
	Controllers Controllers
}
