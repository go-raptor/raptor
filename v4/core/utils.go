package core

import (
	"log/slog"
	"os"
	"strings"

	"github.com/go-raptor/config"
	"github.com/go-raptor/connectors"
	"github.com/pwntr/tinter"
)

type Utils struct {
	Config *config.Config

	Log      *slog.Logger
	LogLevel *slog.LevelVar

	DB connectors.DatabaseConnector
}

func NewUtils() *Utils {
	levelVar := &slog.LevelVar{}

	opts := &tinter.Options{
		Level: levelVar,
	}

	return &Utils{
		Log:      slog.New(tinter.NewHandler(os.Stderr, opts)),
		LogLevel: levelVar,
	}
}

func (u *Utils) SetDB(db connectors.DatabaseConnector) {
	u.DB = db
}

func (u *Utils) SetConfig(config *config.Config) {
	u.Config = config
	u.SetLogLevel(config.GeneralConfig.LogLevel)
}

func (u *Utils) SetLogLevel(logLevel string) {
	u.LogLevel.Set(ParseLogLevel(logLevel))
}

func (u *Utils) SetHandler(handler slog.Handler) {
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
