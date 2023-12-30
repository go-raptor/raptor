package raptor

import (
	"log/slog"
	"os"

	"github.com/lmittmann/tint"
)

type Utils struct {
	Log *slog.Logger
}

func newUtils() *Utils {
	return &Utils{
		Log: slog.New(tint.NewHandler(os.Stderr, nil)),
	}
}
