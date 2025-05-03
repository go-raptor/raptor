package core

import (
	"log/slog"
	"os"
	"strings"

	"github.com/go-raptor/connectors"
	"github.com/go-raptor/raptor/v4/config"
	"github.com/pwntr/tinter"
)

type Resources struct {
	Config *config.Config

	Log      *slog.Logger
	LogLevel *slog.LevelVar

	DB connectors.DatabaseConnector
}

func NewResources() *Resources {
	levelVar := &slog.LevelVar{}

	opts := &tinter.Options{
		Level: levelVar,
	}

	return &Resources{
		Log:      slog.New(tinter.NewHandler(os.Stderr, opts)),
		LogLevel: levelVar,
	}
}

func (u *Resources) SetDB(db connectors.DatabaseConnector) {
	u.DB = db
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
