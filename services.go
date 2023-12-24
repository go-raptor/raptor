package raptor

import (
	"log/slog"
	"os"

	"github.com/lmittmann/tint"
)

type Services struct {
	Log *slog.Logger
}

func newServices() *Services {
	return &Services{
		Log: slog.New(tint.NewHandler(os.Stderr, nil)),
	}
}
