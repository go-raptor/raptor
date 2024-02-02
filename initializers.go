package raptor

type AppInitializer struct {
	Database    Migrations
	Middlewares Middlewares
	Services    Services
	Controllers Controllers
}
