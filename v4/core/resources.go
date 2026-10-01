package core

import (
	"context"
	"log/slog"
	"os"
	"strings"
	"sync/atomic"

	"github.com/go-raptor/connectors"
	"github.com/go-raptor/raptor/v4/config"
)

type Resources struct {
	Config *config.Config

	Log      *slog.Logger
	LogLevel *slog.LevelVar

	Database connectors.DatabaseConnector

	appCtx       context.Context
	cancelApp    context.CancelFunc
	shuttingDown atomic.Bool
}

func NewResources() *Resources {
	levelVar := &slog.LevelVar{}
	appCtx, cancelApp := context.WithCancel(context.Background())

	return &Resources{
		Log:       slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: levelVar})),
		LogLevel:  levelVar,
		appCtx:    appCtx,
		cancelApp: cancelApp,
	}
}

// AppContext is cancelled when the app shuts down, after in-flight requests
// have drained and before services clean up. Background work and database
// calls that don't belong to a request use it, so they stop instead of
// holding shutdown.
func (u *Resources) AppContext() context.Context {
	if u.appCtx == nil {
		return context.Background()
	}
	return u.appCtx
}

// CancelAppContext cancels res's app context; a call on resources without
// one, or a second call, does nothing. Shutdown reaches it through
// Core.CancelAppContext, tests through raptor.CancelAppContext.
func CancelAppContext(res *Resources) {
	if res.cancelApp != nil {
		res.cancelApp()
	}
}

// ShuttingDown reports whether shutdown has begun. Readiness checks use it
// to stop new traffic while requests drain.
func (u *Resources) ShuttingDown() bool {
	return u.shuttingDown.Load()
}

func (u *Resources) SetDB(db connectors.DatabaseConnector) {
	u.Database = db
}

func (u *Resources) SetConfig(config *config.Config) {
	u.Config = config
	u.SetLogLevel(config.GeneralConfig.LogLevel)
}

func (u *Resources) SetLogLevel(logLevel string) {
	u.LogLevel.Set(ParseLogLevel(logLevel))
}

func (u *Resources) SetLogHandler(handler slog.Handler) {
	u.Log = slog.New(handler)
}

func ParseLogLevel(logLevel string) slog.Level {
	var level slog.Level
	switch strings.ToUpper(logLevel) {
	case "DEBUG":
		level = slog.LevelDebug
	case "INFO":
		level = slog.LevelInfo
	case "WARN":
		level = slog.LevelWarn
	case "ERROR":
		level = slog.LevelError
	default:
		level = slog.LevelInfo
	}
	return level
}
