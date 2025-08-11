package core

import (
	"net/http"
	"os"
	"path/filepath"

	"github.com/go-raptor/raptor/v4/errs"
)

func (c *Context) File(file string) error {
	cleanPath := filepath.Clean(file)
	if cleanPath == "." || cleanPath == "/" {
		return errs.ErrNotFound
	}

	f, err := os.Open(cleanPath)
	if err != nil {
		if os.IsNotExist(err) {
			return c.NotFound()
		}
		return errs.NewErrorInternal("Failed to open file", err)
	}
	defer f.Close()

	fi, err := f.Stat()
	if err != nil {
		return errs.NewErrorInternal("Failed to stat file", err)
	}
	if fi.IsDir() {
		return c.NotFound()
	}

	http.ServeContent(c.Response(), c.Request(), fi.Name(), fi.ModTime(), f)
	return nil
}
