package core

import (
	"errors"
	"io/fs"
	"net/http"
	"os"
	"path/filepath"

	"github.com/go-raptor/raptor/v4/errs"
)

// File serves the file at the given path exactly as provided. It offers no
// containment, so never build the path from untrusted input — for anything
// derived from the request, use FileFromDir or FileFromRoot instead.
//
// A missing path, a directory or anything but a regular file returns
// errs.ErrNotFound, which Raptor renders as a JSON 404 without the caching
// headers the handler may have set for a successful response.
func (c *Context) File(file string) error {
	cleanPath := filepath.Clean(file)
	if cleanPath == "." || cleanPath == "/" {
		return errs.ErrNotFound
	}

	// Stat before opening: open(2) on a FIFO blocks until a writer appears.
	fi, err := os.Stat(cleanPath)
	if errors.Is(err, fs.ErrNotExist) {
		return errs.ErrNotFound
	}
	if err != nil {
		return errs.NewErrorInternal("Failed to stat file").WithCause(err)
	}
	if !fi.Mode().IsRegular() {
		return errs.ErrNotFound
	}

	f, err := os.Open(cleanPath)
	if errors.Is(err, fs.ErrNotExist) {
		return errs.ErrNotFound
	}
	if err != nil {
		return errs.NewErrorInternal("Failed to open file").WithCause(err)
	}
	defer f.Close()
	return c.serveFile(f)
}

// FileFromDir serves name from within dir, rejecting anything that escapes
// it — .. traversal, absolute paths, and symlinks pointing outside — via
// os.Root. Safe to use with request-derived names. It opens dir on every
// call; a handler serving many files should open the root once and use
// FileFromRoot.
func (c *Context) FileFromDir(dir, name string) error {
	root, err := os.OpenRoot(dir)
	if err != nil {
		return errs.NewErrorInternal("Failed to open root directory").WithCause(err)
	}
	defer root.Close()
	return c.FileFromRoot(root, name)
}

// FileFromRoot is FileFromDir for a root opened once, typically in a
// controller's Setup; os.Root is safe for concurrent use. Any name that
// does not resolve to a regular file inside root, escapes included, returns
// errs.ErrNotFound.
func (c *Context) FileFromRoot(root *os.Root, name string) error {
	// Stat before opening: open(2) on a FIFO blocks until a writer appears.
	fi, err := root.Stat(name)
	if err != nil || !fi.Mode().IsRegular() {
		return errs.ErrNotFound
	}
	f, err := root.Open(name)
	if err != nil {
		return errs.ErrNotFound
	}
	defer f.Close()
	return c.serveFile(f)
}

// serveFile answers with f's content, handling Range and conditional
// requests. It checks the opened file again, so a directory or device
// swapped in after the first stat is not served either.
func (c *Context) serveFile(f *os.File) error {
	fi, err := f.Stat()
	if err != nil {
		return errs.NewErrorInternal("Failed to stat file").WithCause(err)
	}
	if !fi.Mode().IsRegular() {
		return errs.ErrNotFound
	}
	http.ServeContent(c.Response(), c.Request(), fi.Name(), fi.ModTime(), f)
	return nil
}
