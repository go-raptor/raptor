package core_test

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/go-raptor/raptor/v4/core"
	"github.com/go-raptor/raptor/v4/errs"
)

func newFileContext(t *testing.T) (*core.Context, *httptest.ResponseRecorder) {
	t.Helper()
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/file", nil)
	return core.NewContext(newTestCore(), req, rec), rec
}

func mustWriteFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

func TestFileFromDirServesFile(t *testing.T) {
	dir := t.TempDir()
	mustWriteFile(t, filepath.Join(dir, "hello.txt"), "hello file")
	ctx, rec := newFileContext(t)

	if err := ctx.FileFromDir(dir, "hello.txt"); err != nil {
		t.Fatalf("FileFromDir: %v", err)
	}
	if rec.Code != http.StatusOK {
		t.Fatalf("got %d, want 200", rec.Code)
	}
	if rec.Body.String() != "hello file" {
		t.Fatalf("body: %q", rec.Body.String())
	}
}

func TestFileFromDirServesNestedFile(t *testing.T) {
	dir := t.TempDir()
	mustWriteFile(t, filepath.Join(dir, "sub", "nested.txt"), "nested")
	ctx, rec := newFileContext(t)

	if err := ctx.FileFromDir(dir, "sub/nested.txt"); err != nil {
		t.Fatalf("FileFromDir: %v", err)
	}
	if rec.Code != http.StatusOK || rec.Body.String() != "nested" {
		t.Fatalf("got %d %q", rec.Code, rec.Body.String())
	}
}

func TestFileFromDirBlocksTraversal(t *testing.T) {
	base := t.TempDir()
	dir := filepath.Join(base, "public")
	mustWriteFile(t, filepath.Join(dir, "ok.txt"), "ok")
	mustWriteFile(t, filepath.Join(base, "secret.txt"), "top secret")
	ctx, rec := newFileContext(t)

	err := ctx.FileFromDir(dir, "../secret.txt")

	if strings.Contains(rec.Body.String(), "top secret") {
		t.Fatalf("path traversal escaped the root: %q", rec.Body.String())
	}
	if !errors.Is(err, errs.ErrNotFound) {
		t.Fatalf("got %v, want errs.ErrNotFound", err)
	}
}

func TestFileFromDirBlocksAbsolutePath(t *testing.T) {
	dir := t.TempDir()
	target := filepath.Join(t.TempDir(), "outside.txt")
	mustWriteFile(t, target, "outside")
	ctx, rec := newFileContext(t)

	err := ctx.FileFromDir(dir, target)

	if strings.Contains(rec.Body.String(), "outside") {
		t.Fatalf("absolute path escaped the root: %q", rec.Body.String())
	}
	if !errors.Is(err, errs.ErrNotFound) {
		t.Fatalf("got %v, want errs.ErrNotFound", err)
	}
}

func TestFileFromDirMissingIsNotFoundError(t *testing.T) {
	ctx, rec := newFileContext(t)

	err := ctx.FileFromDir(t.TempDir(), "missing.txt")
	if !errors.Is(err, errs.ErrNotFound) {
		t.Fatalf("got %v, want errs.ErrNotFound", err)
	}
	if ctx.Response().Committed || rec.Body.Len() != 0 {
		t.Fatal("a missing file must be left to Raptor's error rendering, not answered with a bare 404")
	}
}

func TestFileFromDirDirectoryIsNotFoundError(t *testing.T) {
	dir := t.TempDir()
	mustWriteFile(t, filepath.Join(dir, "sub", "x.txt"), "x")
	ctx, _ := newFileContext(t)

	if err := ctx.FileFromDir(dir, "sub"); !errors.Is(err, errs.ErrNotFound) {
		t.Fatalf("got %v, want errs.ErrNotFound", err)
	}
}

func TestFileMissingIsNotFoundError(t *testing.T) {
	ctx, _ := newFileContext(t)

	if err := ctx.File(filepath.Join(t.TempDir(), "missing.txt")); !errors.Is(err, errs.ErrNotFound) {
		t.Fatalf("got %v, want errs.ErrNotFound", err)
	}
}

func TestFileFromRootServesAndConfines(t *testing.T) {
	base := t.TempDir()
	dir := filepath.Join(base, "public")
	mustWriteFile(t, filepath.Join(dir, "ok.txt"), "ok")
	mustWriteFile(t, filepath.Join(base, "secret.txt"), "top secret")
	if err := os.Symlink(filepath.Join(base, "secret.txt"), filepath.Join(dir, "link.txt")); err != nil {
		t.Fatal(err)
	}
	root, err := os.OpenRoot(dir)
	if err != nil {
		t.Fatal(err)
	}
	defer root.Close()

	ctx, rec := newFileContext(t)
	if err := ctx.FileFromRoot(root, "ok.txt"); err != nil || rec.Body.String() != "ok" {
		t.Fatalf("FileFromRoot(ok.txt): %v, body %q", err, rec.Body.String())
	}

	for _, name := range []string{"../secret.txt", "link.txt", filepath.Join(base, "secret.txt")} {
		ctx, rec := newFileContext(t)
		err := ctx.FileFromRoot(root, name)
		if strings.Contains(rec.Body.String(), "top secret") || !errors.Is(err, errs.ErrNotFound) {
			t.Fatalf("FileFromRoot(%q) escaped or erred wrongly: %v, body %q", name, err, rec.Body.String())
		}
	}
}
