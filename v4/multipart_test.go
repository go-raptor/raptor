package raptor_test

import (
	"bytes"
	"context"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/go-raptor/raptor/v4"
	"github.com/go-raptor/raptor/v4/router"
)

type UploadController struct {
	raptor.Controller
}

func (c *UploadController) Parse(ctx *raptor.Context) error {
	// maxMemory 1 sends every file part to a temp file.
	if err := ctx.Request().ParseMultipartForm(1); err != nil {
		return err
	}
	return ctx.NoContent()
}

type swapKey struct{}

// RequestSwapMiddleware replaces the request the way any middleware adding
// a context value does.
type RequestSwapMiddleware struct {
	raptor.Middleware
}

func (m *RequestSwapMiddleware) Handle(ctx *raptor.Context, next func(*raptor.Context) error) error {
	req := ctx.Request()
	ctx.SetRequest(req.WithContext(context.WithValue(req.Context(), swapKey{}, true)))
	return next(ctx)
}

// net/http removes only the form on the request it passed in, so a form
// parsed on a replacement must be removed by Raptor.
func TestMultipartTempFilesRemovedAfterRequestSwap(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("TMPDIR", tmp)
	app := raptor.NewTestApp(
		&raptor.Components{
			Controllers: raptor.Controllers{&UploadController{}},
			Middlewares: raptor.Middlewares{raptor.Use(&RequestSwapMiddleware{})},
		},
		router.Post("/upload", "Upload.Parse"),
	)
	srv := httptest.NewServer(app)
	defer srv.Close()

	var body bytes.Buffer
	form := multipart.NewWriter(&body)
	part, err := form.CreateFormFile("file", "upload.bin")
	if err != nil {
		t.Fatal(err)
	}
	part.Write([]byte("payload"))
	form.Close()

	resp, err := http.Post(srv.URL+"/upload", form.FormDataContentType(), &body)
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusNoContent {
		t.Fatalf("got %d, want 204", resp.StatusCode)
	}

	entries, err := os.ReadDir(tmp)
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 0 {
		t.Fatalf("%d multipart temp file(s) left behind, first %q", len(entries), entries[0].Name())
	}
}
