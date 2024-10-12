package raptor

import (
	"log/slog"
	"os"

	"github.com/lmittmann/tint"
)

type Utils struct {
	Config *Config
	Log    *slog.Logger
	DB     DatabaseConnector
}

func newUtils() *Utils {
	return &Utils{
		Log: slog.New(tint.NewHandler(os.Stderr, nil)),
	}
}

func (u *Utils) SetDB(db DatabaseConnector) {
	u.DB = db
}

func (u *Utils) SetConfig(config *Config) {
	u.Config = config
}
