package raptor_test

import (
	"bytes"
	"errors"
	"log/slog"
	"net/http"
	"strings"
	"testing"

	"github.com/go-raptor/raptor/v4"
	"github.com/go-raptor/raptor/v4/config"
	"github.com/go-raptor/raptor/v4/errs"
	"github.com/go-raptor/raptor/v4/router"
)

type FaultController struct {
	raptor.Controller
}

func (c *FaultController) Boom(ctx *raptor.Context) error {
	return errors.New("secret-db-detail: connection to 10.0.0.5 refused")
}

func (c *FaultController) Deliberate(ctx *raptor.Context) error {
	return errs.NewErrorTeapot("deliberately a teapot")
}

func (c *FaultController) Panics(ctx *raptor.Context) error {
	panic("secret-panic-detail: token abc123")
}

func (c *FaultController) Abort(ctx *raptor.Context) error {
	panic(http.ErrAbortHandler)
}

func (c *FaultController) BindRaw(ctx *raptor.Context) error {
	var v map[string]any
	if err := ctx.Bind(&v); err != nil {
		return err
	}
	return ctx.Status(http.StatusOK)
}

func (c *FaultController) Form(ctx *raptor.Context) error {
	if _, err := ctx.FormParams(); err != nil {
		return err
	}
	return ctx.Status(http.StatusOK)
}

func (c *FaultController) UnencodableAttrs(ctx *raptor.Context) error {
	// errors.New values have no exported fields, which encoding/json/v2 refuses.
	return errs.NewErrorUnprocessableEntity("Validation failed", "cause", errors.New("boom"))
}

func (c *FaultController) InvalidUTF8(ctx *raptor.Context) error {
	return errs.NewErrorBadRequest("bad byte \xff")
}

func newFaultApp(logBuf *bytes.Buffer, opts ...raptor.RaptorOption) *raptor.Raptor {
	if logBuf != nil {
		opts = append(opts, raptor.WithLogHandler(func(level *slog.LevelVar) slog.Handler {
			return slog.NewTextHandler(logBuf, &slog.HandlerOptions{Level: level})
		}))
	}
	return raptor.NewTestApp(
		&raptor.Components{Controllers: raptor.Controllers{&FaultController{}}},
		router.CollectRoutes(
			router.Get("/boom", "Fault.Boom"),
			router.Get("/teapot", "Fault.Deliberate"),
			router.Get("/panic", "Fault.Panics"),
			router.Get("/abort", "Fault.Abort"),
			router.Post("/bind", "Fault.BindRaw"),
			router.Post("/form", "Fault.Form"),
			router.Get("/unencodable", "Fault.UnencodableAttrs"),
			router.Get("/invalid-utf8", "Fault.InvalidUTF8"),
		),
		opts...,
	)
}

func TestInternalErrorRedactedFromClient(t *testing.T) {
	var logBuf bytes.Buffer
	app := newFaultApp(&logBuf)

	rec := app.TestGet("/boom")
	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("GET /boom: got %d, want 500", rec.Code)
	}
	if body := rec.Body.String(); strings.Contains(body, "secret-db-detail") {
		t.Fatalf("internal error details leaked to the client: %s", body)
	}
	if body := rec.Body.String(); !strings.Contains(body, "Internal Server Error") {
		t.Fatalf("client should get a generic 500 body: %s", body)
	}
	if !strings.Contains(logBuf.String(), "secret-db-detail") {
		t.Fatal("internal error detail should still be logged server-side")
	}
}

func TestErrsErrorMessagePreserved(t *testing.T) {
	app := newFaultApp(nil)

	rec := app.TestGet("/teapot")
	if rec.Code != http.StatusTeapot {
		t.Fatalf("GET /teapot: got %d, want 418", rec.Code)
	}
	if !strings.Contains(rec.Body.String(), "deliberately a teapot") {
		t.Fatalf("deliberate errs.Error messages must reach the client: %s", rec.Body.String())
	}
}

func TestPanicRedactedAndLoggedWithStack(t *testing.T) {
	var logBuf bytes.Buffer
	app := newFaultApp(&logBuf)

	rec := app.TestGet("/panic")
	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("GET /panic: got %d, want 500", rec.Code)
	}
	if body := rec.Body.String(); strings.Contains(body, "secret-panic-detail") {
		t.Fatalf("panic value leaked to the client: %s", body)
	}
	log := logBuf.String()
	if !strings.Contains(log, "secret-panic-detail") {
		t.Fatal("panic value should be logged server-side")
	}
	if !strings.Contains(log, "goroutine") {
		t.Fatal("panic log should include a stack trace")
	}
}

func TestAbortHandlerPanicIsRepanicked(t *testing.T) {
	app := newFaultApp(nil)

	panicked := false
	func() {
		defer func() {
			if rec := recover(); rec != nil {
				panicked = true
				if err, ok := rec.(error); !ok || !errors.Is(err, http.ErrAbortHandler) {
					t.Errorf("re-panicked with wrong value: %v", rec)
				}
			}
		}()
		app.TestGet("/abort")
	}()
	if !panicked {
		t.Fatal("http.ErrAbortHandler was swallowed; the stdlib contract requires re-panicking it")
	}
}

func TestBindOverLimitReturns413(t *testing.T) {
	app := newFaultApp(nil, raptor.WithConfig(&config.Config{
		ServerConfig: config.ServerConfig{MaxBodyBytes: 16},
	}))

	body := strings.NewReader(`{"a":"` + strings.Repeat("x", 100) + `"}`)
	rec := app.TestPost("/bind", body)
	if rec.Code != http.StatusRequestEntityTooLarge {
		t.Fatalf("oversized body: got %d, want 413", rec.Code)
	}
}

func TestFormOverLimitReturns413(t *testing.T) {
	app := newFaultApp(nil, raptor.WithConfig(&config.Config{
		ServerConfig: config.ServerConfig{MaxBodyBytes: 16},
	}))

	body := strings.NewReader("a=" + strings.Repeat("x", 100))
	rec := app.TestPost("/form", body,
		raptor.WithHeader("Content-Type", "application/x-www-form-urlencoded"))
	if rec.Code != http.StatusRequestEntityTooLarge {
		t.Fatalf("oversized form body should be limited centrally, not only in Bind: got %d, want 413", rec.Code)
	}
}

func TestDefaultMaxBodyBytes(t *testing.T) {
	if got := config.NewConfigDefaults().ServerConfig.MaxBodyBytes; got != 8<<20 {
		t.Fatalf("default MaxBodyBytes: got %d, want %d (8 MB)", got, int64(8<<20))
	}
}

func TestUnencodableAttrsAreDroppedNot200(t *testing.T) {
	var logBuf bytes.Buffer
	app := newFaultApp(&logBuf)

	rec := app.TestGet("/unencodable")
	if rec.Code == http.StatusOK {
		t.Fatal("an error response that fails to encode must never answer 200")
	}
	if rec.Code != http.StatusUnprocessableEntity {
		t.Fatalf("got %d, want 422", rec.Code)
	}
	if got, want := rec.Body.String(), `{"code":422,"message":"Validation failed"}`; got != want {
		t.Fatalf("body = %s, want %s", got, want)
	}
	if !strings.Contains(logBuf.String(), "Failed to encode error response") {
		t.Fatalf("the encoding failure must be logged: %s", logBuf.String())
	}
}

func TestInvalidUTF8MessageKeepsStatus(t *testing.T) {
	app := newFaultApp(nil)

	rec := app.TestGet("/invalid-utf8")
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("got %d, want 400: an unencodable message must not turn the error into a 500", rec.Code)
	}
	if got, want := rec.Body.String(), "{\"code\":400,\"message\":\"bad byte \ufffd\"}"; got != want {
		t.Fatalf("body = %s, want %s", got, want)
	}
	if ct := rec.Header().Get("Content-Type"); !strings.HasPrefix(ct, "application/json") {
		t.Fatalf("Content-Type = %q, want application/json", ct)
	}
}

func TestNotFoundPathWithInvalidUTF8Is404(t *testing.T) {
	app := newFaultApp(nil)

	// Scanners send paths like this; the built-in NotFound message echoes the path.
	rec := app.TestGet("/nope%ff")
	if rec.Code != http.StatusNotFound || !strings.Contains(rec.Body.String(), `"code":404`) {
		t.Fatalf("got %d %s, want a JSON 404", rec.Code, rec.Body)
	}
}
