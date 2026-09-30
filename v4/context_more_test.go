package raptor_test

import (
	"encoding/json/v2"
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

var leakedContext *raptor.Context

func (c *StateController) Leak(ctx *raptor.Context) error {
	ctx.Set("user", "alice")
	leakedContext = ctx
	return ctx.Status(http.StatusOK)
}

func TestFinishedContextKeepsNoRequestData(t *testing.T) {
	app := raptor.NewTestApp(
		&raptor.Components{Controllers: raptor.Controllers{&StateController{}}},
		router.CollectRoutes(router.Get("/leak", "State.Leak")),
	)

	app.TestGet("/leak?q=1")
	if leakedContext.Request() != nil {
		t.Fatal("a pooled context still references the finished request")
	}
	if leakedContext.Get("user") != nil {
		t.Fatal("a pooled context still holds the finished request's stored values")
	}
}

func (c *StateController) IP(ctx *raptor.Context) error {
	first, second := ctx.RealIP(), ctx.RealIP()
	swapped := ctx.Request().Clone(ctx.Request().Context())
	swapped.RemoteAddr = "198.51.100.7:1234"
	ctx.SetRequest(swapped)
	return ctx.String(http.StatusOK, first+" "+second+" "+ctx.RealIP())
}

func TestRealIPComputedOncePerRequest(t *testing.T) {
	app := raptor.NewTestApp(
		&raptor.Components{Controllers: raptor.Controllers{&StateController{}}},
		router.CollectRoutes(router.Get("/ip", "State.IP")),
	)
	calls := 0
	extract := app.Core.IPExtractor
	app.Core.IPExtractor = func(r *http.Request) string {
		calls++
		return extract(r)
	}

	rec := app.TestGet("/ip", raptor.WithRemoteAddr("203.0.113.9"))
	if got, want := rec.Body.String(), "203.0.113.9 203.0.113.9 198.51.100.7"; got != want {
		t.Fatalf("body %q, want %q: a replaced request must not reuse the old address", got, want)
	}
	if calls != 2 {
		t.Fatalf("extractor ran %d times, want 2: once per request value", calls)
	}
}

type ParamsController struct {
	raptor.Controller
}

func (c *ParamsController) Show(ctx *raptor.Context) error {
	id, err := ctx.ParamInt64("id")
	if err != nil {
		return err
	}
	course, err := ctx.QueryInt64("courseId")
	if err != nil {
		return err
	}
	return ctx.Data(map[string]int64{"id": id, "course": course})
}

func TestTypedParams(t *testing.T) {
	app := raptor.NewTestApp(
		&raptor.Components{Controllers: raptor.Controllers{&ParamsController{}}},
		router.CollectRoutes(router.Get("/things/{id}", "Params.Show")),
	)

	// json/v2 doesn't sort map keys, so compare values, not bytes.
	rec := app.TestGet("/things/42?courseId=-7")
	var got map[string]int64
	if err := json.Unmarshal(rec.Body.Bytes(), &got); rec.Code != http.StatusOK || err != nil || got["id"] != 42 || got["course"] != -7 {
		t.Fatalf("valid params: %d %s (%v)", rec.Code, rec.Body, err)
	}
	for _, path := range []string{
		"/things/abc?courseId=1",
		"/things/99999999999999999999?courseId=1",
		"/things/1.5?courseId=1",
		"/things/42",
		"/things/42?courseId=",
		"/things/42?courseId=x",
	} {
		if rec := app.TestGet(path); rec.Code != http.StatusBadRequest {
			t.Errorf("%s: got %d %s, want 400", path, rec.Code, rec.Body)
		}
	}
}
