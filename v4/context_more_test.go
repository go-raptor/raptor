package raptor_test

import (
	"net/http"
	"strconv"
	"strings"
	"testing"

	"github.com/go-raptor/raptor/v4"
	"github.com/go-raptor/raptor/v4/errs"
	"github.com/go-raptor/raptor/v4/router"
)

func TestLargeJSONResponseSetsContentLength(t *testing.T) {
	app := raptor.NewTestApp(
		&raptor.Components{Controllers: raptor.Controllers{&StateController{}}},
		router.CollectRoutes(router.Get("/big", "State.Big")),
	)

	rec := app.TestGet("/big")
	if rec.Code != http.StatusOK {
		t.Fatalf("got %d, want 200", rec.Code)
	}
	if rec.Body.Len() <= 2048 {
		t.Fatalf("test body must exceed the chunking threshold, got %d bytes", rec.Body.Len())
	}
	want := strconv.Itoa(rec.Body.Len())
	if got := rec.Header().Get("Content-Length"); got != want {
		t.Fatalf("large responses must carry Content-Length to avoid chunked encoding: got %q, want %q", got, want)
	}
}

type StateController struct {
	raptor.Controller
}

func (c *StateController) SetVal(ctx *raptor.Context) error {
	ctx.Set("k", "v")
	return ctx.Status(http.StatusOK)
}

func (c *StateController) GetVal(ctx *raptor.Context) error {
	if ctx.Get("k") != nil {
		return errs.NewErrorInternal("request-scoped store leaked across pooled contexts")
	}
	return ctx.Status(http.StatusOK)
}

func (c *StateController) Big(ctx *raptor.Context) error {
	return ctx.Data(map[string]string{"data": strings.Repeat("x", 4096)})
}

func TestContextStoreIsolatedBetweenRequests(t *testing.T) {
	app := raptor.NewTestApp(
		&raptor.Components{Controllers: raptor.Controllers{&StateController{}}},
		router.CollectRoutes(
			router.Get("/set", "State.SetVal"),
			router.Get("/get", "State.GetVal"),
		),
	)

	if rec := app.TestGet("/set"); rec.Code != http.StatusOK {
		t.Fatalf("GET /set: got %d", rec.Code)
	}
	if rec := app.TestGet("/get"); rec.Code != http.StatusOK {
		t.Fatalf("GET /get: got %d — pooled context state leaked", rec.Code)
	}
}

type TagMiddleware struct {
	raptor.Middleware

	tag string
	log *[]string
}

func (m *TagMiddleware) Handle(ctx *raptor.Context, next func(*raptor.Context) error) error {
	*m.log = append(*m.log, m.tag)
	return next(ctx)
}

func TestMiddlewareScopingAndOrder(t *testing.T) {
	var calls []string
	app := raptor.NewTestApp(
		&raptor.Components{
			Controllers: raptor.Controllers{&RoutesController{}},
			Middlewares: raptor.Middlewares{
				raptor.Use(&TagMiddleware{tag: "global", log: &calls}),
				raptor.UseOnly(&TagMiddleware{tag: "only-show", log: &calls}, "Routes.Show"),
				raptor.UseExcept(&TagMiddleware{tag: "except-show", log: &calls}, "Routes.Show"),
			},
		},
		router.CollectRoutes(
			router.Get("/things/{id}", "Routes.Show"),
			router.Get("/hello", "Routes.Hello"),
		),
	)

	calls = nil
	app.TestGet("/things/1")
	if got := append([]string(nil), calls...); len(got) != 2 || got[0] != "global" || got[1] != "only-show" {
		t.Fatalf("Show chain: got %v, want [global only-show]", got)
	}

	calls = nil
	app.TestGet("/hello")
	if got := append([]string(nil), calls...); len(got) != 2 || got[0] != "global" || got[1] != "except-show" {
		t.Fatalf("Hello chain: got %v, want [global except-show]", got)
	}

	calls = nil
	app.TestGet("/nope")
	if got := append([]string(nil), calls...); len(got) != 2 || got[0] != "global" || got[1] != "except-show" {
		t.Fatalf("404 chain: got %v, want [global except-show] (middleware wraps error handlers)", got)
	}
}
