package raptor_test

import (
	"bytes"
	"encoding/json/jsontext"
	"encoding/json/v2"
	"errors"
	"log/slog"
	"net/http"
	"strings"
	"testing"

	"github.com/go-raptor/raptor/v4"
	"github.com/go-raptor/raptor/v4/errs"
	"github.com/go-raptor/raptor/v4/router"
)

type bindPayload struct {
	Name string `json:"name"`
}

type BindController struct {
	raptor.Controller
}

func (c *BindController) Strict(ctx *raptor.Context) error {
	var p bindPayload
	if err := ctx.BindWith(&p, json.RejectUnknownMembers(true)); err != nil {
		if errors.Is(err, json.ErrUnknownName) {
			return errs.NewErrorBadRequest("unknown field")
		}
		return errs.NewErrorBadRequest("invalid JSON")
	}
	return ctx.Data(p)
}

func (c *BindController) Lenient(ctx *raptor.Context) error {
	var p bindPayload
	if err := ctx.Bind(&p); err != nil {
		return errs.NewErrorBadRequest("invalid JSON")
	}
	return ctx.Data(p)
}

func (c *BindController) Raw(ctx *raptor.Context) error {
	var p bindPayload
	if err := ctx.Bind(&p); err != nil {
		return err
	}
	return ctx.Data(p)
}

func (c *BindController) Cause(ctx *raptor.Context) error {
	var p bindPayload
	err := ctx.Bind(&p)
	if _, ok := errors.AsType[*jsontext.SyntacticError](err); !ok {
		return errs.NewErrorInternal("the decode error is no longer reachable")
	}
	return ctx.NoContent()
}

func newBindApp(logBuf *bytes.Buffer) *raptor.Raptor {
	var opts []raptor.RaptorOption
	if logBuf != nil {
		opts = append(opts, raptor.WithLogHandler(func(level *slog.LevelVar) slog.Handler {
			return slog.NewTextHandler(logBuf, &slog.HandlerOptions{Level: level})
		}))
	}
	return raptor.NewTestApp(
		&raptor.Components{Controllers: raptor.Controllers{&BindController{}}},
		router.CollectRoutes(
			router.Post("/raw", "Bind.Raw"),
			router.Post("/cause", "Bind.Cause"),
		),
		opts...,
	)
}

func TestBindWithRejectsUnknownMembers(t *testing.T) {
	app := raptor.NewTestApp(
		&raptor.Components{Controllers: raptor.Controllers{&BindController{}}},
		router.CollectRoutes(
			router.Post("/strict", "Bind.Strict"),
			router.Post("/lenient", "Bind.Lenient"),
		),
	)
	body := `{"name":"a","extra":1}`

	rec := app.TestPost("/strict", strings.NewReader(body))
	if rec.Code != http.StatusBadRequest || !strings.Contains(rec.Body.String(), "unknown field") {
		t.Fatalf("BindWith(RejectUnknownMembers) must fail with json.ErrUnknownName: %d %s", rec.Code, rec.Body)
	}
	rec = app.TestPost("/lenient", strings.NewReader(body))
	if rec.Code != http.StatusOK || rec.Body.String() != `{"name":"a"}` {
		t.Fatalf("Bind must keep ignoring unknown members: %d %s", rec.Code, rec.Body)
	}
}

func TestBindMalformedJSONIs400(t *testing.T) {
	var logBuf bytes.Buffer
	app := newBindApp(&logBuf)

	for _, body := range []string{`{"name":`, ``, `[1,2]`, `{"name":"a"} trailing`} {
		rec := app.TestPost("/raw", strings.NewReader(body))
		if rec.Code != http.StatusBadRequest {
			t.Errorf("%q: got %d, want 400", body, rec.Code)
		}
	}
	if strings.Contains(logBuf.String(), "Unhandled error") {
		t.Fatalf("a client's malformed JSON is not a server error: %s", logBuf.String())
	}
}

func TestBindRequiresJSONContentType(t *testing.T) {
	app := newBindApp(nil)

	for _, ct := range []string{"text/plain", "application/x-www-form-urlencoded", "", "application/jsonx", "text/json"} {
		rec := app.TestPost("/raw", strings.NewReader(`{"name":"a"}`), raptor.WithHeader("Content-Type", ct))
		if rec.Code != http.StatusUnsupportedMediaType {
			t.Errorf("Content-Type %q: got %d, want 415", ct, rec.Code)
		}
	}
	for _, ct := range []string{"application/json", "application/json; charset=utf-8", "Application/JSON", "application/merge-patch+json", "application/vnd.api+json"} {
		rec := app.TestPost("/raw", strings.NewReader(`{"name":"a"}`), raptor.WithHeader("Content-Type", ct))
		if rec.Code != http.StatusOK {
			t.Errorf("Content-Type %q: got %d, want 200", ct, rec.Code)
		}
	}
}

func TestBindErrorKeepsDecodeCause(t *testing.T) {
	app := newBindApp(nil)

	if rec := app.TestPost("/cause", strings.NewReader(`{bad}`)); rec.Code != http.StatusNoContent {
		t.Fatalf("got %d %s: errors.As must still find the jsontext.SyntacticError", rec.Code, rec.Body)
	}
}
