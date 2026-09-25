package raptor_test

import (
	"encoding/json/v2"
	"errors"
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
