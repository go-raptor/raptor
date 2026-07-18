package core

import (
	"net/http"
	"os"
	"path/filepath"

	"github.com/go-raptor/raptor/v4/errs"
)

// File serves the file at the given path exactly as provided. It offers no
// containment, so never build the path from untrusted input — for anything
// derived from the request, use FileFromDir instead.
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
		return errs.NewErrorInternal("Failed to open file").WithCause(err)
	}
	defer f.Close()

	fi, err := f.Stat()
	if err != nil {
		return errs.NewErrorInternal("Failed to stat file").WithCause(err)
	}
	if fi.IsDir() {
		return c.NotFound()
	}

	http.ServeContent(c.Response(), c.Request(), fi.Name(), fi.ModTime(), f)
	return nil
}

// FileFromDir serves name from within dir, rejecting anything that escapes
// it — .. traversal, absolute paths, and symlinks pointing outside — via
// os.Root. Safe to use with request-derived names.
func (c *Context) FileFromDir(dir, name string) error {
	root, err := os.OpenRoot(dir)
	if err != nil {
		return errs.NewErrorInternal("Failed to open root directory").WithCause(err)
	}
	defer root.Close()

	f, err := root.Open(name)
	if err != nil {
		return c.NotFound()
	}
	defer f.Close()

	fi, err := f.Stat()
	if err != nil {
		return errs.NewErrorInternal("Failed to stat file").WithCause(err)
	}
	if fi.IsDir() {
		return c.NotFound()
	}

	http.ServeContent(c.Response(), c.Request(), fi.Name(), fi.ModTime(), f)
	return nil
}
