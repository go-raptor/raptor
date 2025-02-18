package core

import (
	"log/slog"
	"sync"

	"github.com/go-raptor/connector"
	"github.com/go-raptor/raptor/v3/config"
	"github.com/labstack/echo/v4"
)

type Components struct {
	DatabaseConnector connector.DatabaseConnector
	Middlewares       Middlewares
	Services          Services
	Controllers       Controllers
}

type Core struct {
	utils       *Utils
	handlers    map[string]map[string]*handler
	contextPool sync.Pool
	services    map[string]ServiceInterface
	middlewares []MiddlewareInterface
}

func NewCore(u *Utils) *Core {
	return &Core{
		utils:    u,
		handlers: make(map[string]map[string]*handler),
		contextPool: sync.Pool{
			New: func() interface{} {
				return new(Context)
			},
		},
		services:    make(map[string]ServiceInterface),
		middlewares: make([]MiddlewareInterface, 0),
	}
}

type Map map[string]interface{}

type Context struct {
	echo.Context
	Controller string
	Action     string
}

type Controllers []interface{}

type ControllerInterface interface {
	Init(u *Utils)
}

type Controller struct {
	*Utils
	onInit func()
}

type Services []ServiceInterface

type ServiceInterface interface {
	InitService(u *Utils) error
}

type Service struct {
	*Utils
	onInit func() error
}

type ScopedMiddleware struct {
	middleware MiddlewareInterface
	only       []string
	except     []string
	global     bool
}
type Middlewares []ScopedMiddleware

type MiddlewareInterface interface {
	InitMiddleware(u *Utils)
	New(*Context) error
}

type Middleware struct {
	*Utils
	onInit func()
}

type echoMiddleware struct {
	handler echo.HandlerFunc
}

type Utils struct {
	Config *config.Config

	Log      *slog.Logger
	logLevel *slog.LevelVar

	DB connector.DatabaseConnector
}

type Error struct {
	Code        int    `json:"code"`
	Message     string `json:"message"`
	Description string `json:"description,omitempty"`
}
