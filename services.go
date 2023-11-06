package main

import (
	"log/slog"
	"os"

	"github.com/lmittmann/tint"
)

type Services struct {
	Log *slog.Logger
}

func NewServices() *Services {
	return &Services{
		Log: slog.New(tint.NewHandler(os.Stderr, nil)),
	}
}
