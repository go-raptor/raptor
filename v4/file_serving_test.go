package raptor_test

import (
	"mime"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"testing"
	"unicode"

	"github.com/go-raptor/raptor/v4"
	"github.com/go-raptor/raptor/v4/router"
)

type FileServingController struct {
	raptor.Controller
	dir string
}

func (c *FileServingController) Cached(ctx *raptor.Context) error {
	ctx.Response().Header().Set("Cache-Control", "public, max-age=31536000, immutable")
	return ctx.FileFromDir(c.dir, ctx.Param("name"))
}

func (c *FileServingController) DownloadMissing(ctx *raptor.Context) error {
	return ctx.Attachment(filepath.Join(c.dir, "missing.txt"), "report.txt")
}

func (c *FileServingController) Named(ctx *raptor.Context) error {
	return ctx.Attachment(filepath.Join(c.dir, "hello.txt"), ctx.QueryParam("as"))
}

func newFilesApp(t *testing.T) *raptor.Raptor {
	t.Helper()
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "hello.txt"), []byte("hello file"), 0o644); err != nil {
		t.Fatal(err)
	}
	return raptor.NewTestApp(
		&raptor.Components{Controllers: raptor.Controllers{&FileServingController{dir: dir}}},
		router.CollectRoutes(
			router.Get("/cached/{name}", "FileServing.Cached"),
			router.Get("/download-missing", "FileServing.DownloadMissing"),
			router.Get("/named", "FileServing.Named"),
		),
	)
}

func TestMissingFileDropsSuccessCacheHeaders(t *testing.T) {
	app := newFilesApp(t)

	rec := app.TestGet("/cached/missing.txt")
	if rec.Code != http.StatusNotFound {
		t.Fatalf("got %d, want 404", rec.Code)
	}
	if cc := rec.Header().Get("Cache-Control"); cc != "no-store" {
		t.Fatalf("Cache-Control %q on a 404 lets a CDN cache the miss, want no-store", cc)
	}
	if got, want := rec.Body.String(), `{"code":404,"message":"Not Found"}`; got != want {
		t.Fatalf("body %s, want %s", got, want)
	}

	rec = app.TestGet("/cached/hello.txt")
	if rec.Code != http.StatusOK || rec.Header().Get("Cache-Control") == "" {
		t.Fatalf("a served file keeps its caching headers: %d %q", rec.Code, rec.Header().Get("Cache-Control"))
	}
}

func TestMissingAttachmentIsNotADownload(t *testing.T) {
	app := newFilesApp(t)

	rec := app.TestGet("/download-missing")
	if rec.Code != http.StatusNotFound {
		t.Fatalf("got %d, want 404", rec.Code)
	}
	if cd := rec.Header().Get("Content-Disposition"); cd != "" {
		t.Fatalf("Content-Disposition %q makes the browser save the error as a file", cd)
	}
}

func TestFileRangeRequest(t *testing.T) {
	app := newFilesApp(t)

	rec := app.TestGet("/cached/hello.txt", raptor.WithHeader("Range", "bytes=0-4"))
	if rec.Code != http.StatusPartialContent || rec.Body.String() != "hello" {
		t.Fatalf("got %d %q, want 206 %q", rec.Code, rec.Body.String(), "hello")
	}
}

func TestAttachmentFilenameEncoding(t *testing.T) {
	app := newFilesApp(t)

	for _, name := range []string{"report.pdf", `quote "q" \ back.txt`, "čćž ünï.txt", "a\r\nb.txt"} {
		rec := app.TestGet("/named?as=" + url.QueryEscape(name))
		cd := rec.Header().Get("Content-Disposition")
		for _, r := range cd {
			if r > unicode.MaxASCII || r == '\r' || r == '\n' {
				t.Fatalf("%q: header %q must be ASCII without line breaks", name, cd)
			}
		}
		typ, params, err := mime.ParseMediaType(cd)
		if err != nil || typ != "attachment" || params["filename"] != name {
			t.Errorf("%q: header %q parses to %q %q (%v)", name, cd, typ, params["filename"], err)
		}
	}
}
