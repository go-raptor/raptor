package raptor

import (
	"log/slog"
	"os"

	"github.com/lmittmann/tint"
	"gorm.io/gorm"
)

type Utils struct {
	Log *slog.Logger
	DB  *gorm.DB
}

func newUtils() *Utils {
	return &Utils{
		Log: slog.New(tint.NewHandler(os.Stderr, nil)),
	}
}

func (u *Utils) SetDB(db *gorm.DB) {
	u.DB = db
}
