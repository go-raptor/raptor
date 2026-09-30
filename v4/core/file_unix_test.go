//go:build unix

package core_test

import (
	"errors"
	"os"
	"path/filepath"
	"syscall"
	"testing"
	"time"

	"github.com/go-raptor/raptor/v4/core"
	"github.com/go-raptor/raptor/v4/errs"
)

// A FIFO blocks open(2) until a writer appears, so serving one would hang
// the request. Only regular files may be served.
func TestFileServingSkipsFIFO(t *testing.T) {
	dir := t.TempDir()
	fifo := filepath.Join(dir, "pipe")
	if err := syscall.Mkfifo(fifo, 0o644); err != nil {
		t.Skipf("mkfifo: %v", err)
	}

	cases := []struct {
		name  string
		serve func(*core.Context) error
	}{
		{"FileFromDir", func(ctx *core.Context) error { return ctx.FileFromDir(dir, "pipe") }},
		{"File", func(ctx *core.Context) error { return ctx.File(fifo) }},
	}
	for _, tc := range cases {
		ctx, _ := newFileContext(t)
		done := make(chan error, 1)
		go func() { done <- tc.serve(ctx) }()
		select {
		case err := <-done:
			if !errors.Is(err, errs.ErrNotFound) {
				t.Errorf("%s: got %v, want errs.ErrNotFound", tc.name, err)
			}
		case <-time.After(2 * time.Second):
			unblockFIFO(fifo)
			<-done
			t.Fatalf("%s blocked opening a FIFO", tc.name)
		}
	}
}

// unblockFIFO opens the write end so a reader stuck in open(2) returns.
func unblockFIFO(path string) {
	if f, err := os.OpenFile(path, os.O_WRONLY|syscall.O_NONBLOCK, 0); err == nil {
		f.Close()
	}
}
