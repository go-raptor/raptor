package raptor

type AppInitializer struct {
	Middlewares Middlewares
	Services    Services
	Controllers Controllers
}
