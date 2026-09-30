package raptor_test

import (
	"bytes"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/go-raptor/raptor/v4"
	"github.com/go-raptor/raptor/v4/router"
)

type StdController struct {
	raptor.Controller
}

func (c *StdController) Text(ctx *raptor.Context) error {
	return ctx.String(http.StatusOK, "hello")
}

func (c *StdController) Header(ctx *raptor.Context) error {
	return ctx.String(http.StatusOK, ctx.Request().Header.Get("X-From-Middleware"))
}

func (c *StdController) Stream(ctx *raptor.Context) error {
	body := "hello"
	return ctx.Stream(http.StatusOK, "text/plain", io.LimitReader(strings.NewReader(body), int64(len(body))))
}

type upperWriter struct {
	http.ResponseWriter
}

func (w *upperWriter) Write(b []byte) (int, error) {
	return w.ResponseWriter.Write(bytes.ToUpper(b))
}

func newStdApp(mw func(http.Handler) http.Handler) *raptor.Raptor {
	return raptor.NewTestApp(
		&raptor.Components{
			Controllers: raptor.Controllers{&StdController{}},
			Middlewares: raptor.Middlewares{raptor.UseStd(mw)},
		},
		router.CollectRoutes(
			router.Get("/text", "Std.Text"),
			router.Get("/header", "Std.Header"),
			router.Get("/stream", "Std.Stream"),
		),
	)
}

func TestUseStdWriterWrappingMiddleware(t *testing.T) {
	app := newStdApp(func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			next.ServeHTTP(&upperWriter{ResponseWriter: w}, r)
		})
	})

	rec := app.TestGet("/text")
	if rec.Code != http.StatusOK {
		t.Fatalf("GET /text: got %d, want 200", rec.Code)
	}
	if body := rec.Body.String(); body != "HELLO" {
		t.Fatalf("writer-wrapping std middleware was bypassed: body %q, want %q", body, "HELLO")
	}
}

func TestUseStdRequestModifyingMiddleware(t *testing.T) {
	app := newStdApp(func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			r2 := r.Clone(r.Context())
			r2.Header.Set("X-From-Middleware", "present")
			next.ServeHTTP(w, r2)
		})
	})

	rec := app.TestGet("/header")
	if body := rec.Body.String(); body != "present" {
		t.Fatalf("request modification was not propagated: body %q, want %q", body, "present")
	}
}

func TestUseStdShortCircuitMiddleware(t *testing.T) {
	app := newStdApp(func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusTeapot)
		})
	})

	rec := app.TestGet("/text")
	if rec.Code != http.StatusTeapot {
		t.Fatalf("short-circuiting std middleware: got %d, want 418", rec.Code)
	}
}

// A substituted writer without ReadFrom must still see every byte: ReadFrom
// forwards to the writer in front of it, never to the connection beneath.
func TestUseStdWriterSeesCopiedBody(t *testing.T) {
	app := newStdApp(func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			next.ServeHTTP(&upperWriter{ResponseWriter: w}, r)
		})
	})

	rec := app.TestGet("/stream")
	if body := rec.Body.String(); body != "HELLO" {
		t.Fatalf("a copy bypassed the middleware's writer: body %q, want %q", body, "HELLO")
	}
}
