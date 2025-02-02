package raptor

import (
	"log/slog"
	"os"
	"strings"

	"github.com/go-raptor/connector"
	"github.com/pwntr/tinter"
)

type Utils struct {
	Config *Config

	Log      *slog.Logger
	logLevel *slog.LevelVar

	DB connector.DatabaseConnector
}

func newUtils() *Utils {
	levelVar := &slog.LevelVar{}

	opts := &tinter.Options{
		Level: levelVar,
	}

	return &Utils{
		Log:      slog.New(tinter.NewHandler(os.Stderr, opts)),
		logLevel: levelVar,
	}
}

func (u *Utils) SetDB(db connector.DatabaseConnector) {
	u.DB = db
}

func (u *Utils) SetConfig(config *Config) {
	u.Config = config
}

func (u *Utils) SetLogLevel(logLevel string) {
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
	u.logLevel.Set(level)
}
