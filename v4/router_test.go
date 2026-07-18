package raptor_test

import (
	"net/http"
	"testing"

	"github.com/go-raptor/raptor/v4"
	"github.com/go-raptor/raptor/v4/router"
)

type RoutesController struct {
	raptor.Controller
}

func (c *RoutesController) Hello(ctx *raptor.Context) error {
	return ctx.Data(map[string]string{"message": "hello"})
}

func (c *RoutesController) Show(ctx *raptor.Context) error {
	return ctx.Data(map[string]string{"id": ctx.Param("id")})
}

func (c *RoutesController) Create(ctx *raptor.Context) error {
	return ctx.Status(http.StatusCreated)
}

func newRoutesApp(routes router.Routes) *raptor.Raptor {
	return raptor.NewTestApp(
		&raptor.Components{Controllers: raptor.Controllers{&RoutesController{}}},
		routes,
	)
}

func TestParamNameDriftDoesNotPanic(t *testing.T) {
	app := newRoutesApp(router.CollectRoutes(
		router.Get("/things/{id}", "Routes.Show"),
		router.Post("/things/{slug}", "Routes.Create"),
	))

	rec := app.TestGet("/things/42")
	if rec.Code != http.StatusOK {
		t.Fatalf("GET /things/42: got %d, want 200", rec.Code)
	}
	rec = app.TestPost("/things/42", nil)
	if rec.Code != http.StatusCreated {
		t.Fatalf("POST /things/42: got %d, want 201", rec.Code)
	}
}

func TestMethodNotAllowed(t *testing.T) {
	app := newRoutesApp(router.CollectRoutes(
		router.Get("/hello", "Routes.Hello"),
	))

	rec := app.TestPost("/hello", nil)
	if rec.Code != http.StatusMethodNotAllowed {
		t.Fatalf("POST /hello: got %d, want 405", rec.Code)
	}
	if allow := rec.Header().Get("Allow"); allow != "GET, HEAD" {
		t.Fatalf("Allow header: got %q, want %q", allow, "GET, HEAD")
	}
}

func TestMethodNotAllowedMultipleMethods(t *testing.T) {
	app := newRoutesApp(router.CollectRoutes(
		router.Get("/multi", "Routes.Hello"),
		router.Put("/multi", "Routes.Create"),
	))

	rec := app.TestPatch("/multi", nil)
	if rec.Code != http.StatusMethodNotAllowed {
		t.Fatalf("PATCH /multi: got %d, want 405", rec.Code)
	}
	if allow := rec.Header().Get("Allow"); allow != "GET, HEAD, PUT" {
		t.Fatalf("Allow header: got %q, want %q", allow, "GET, HEAD, PUT")
	}
}

func TestNotFound(t *testing.T) {
	app := newRoutesApp(router.CollectRoutes(
		router.Get("/hello", "Routes.Hello"),
	))

	rec := app.TestGet("/nope")
	if rec.Code != http.StatusNotFound {
		t.Fatalf("GET /nope: got %d, want 404", rec.Code)
	}
	rec = app.TestPost("/nope", nil)
	if rec.Code != http.StatusNotFound {
		t.Fatalf("POST /nope: got %d, want 404 (no methods match this path)", rec.Code)
	}
}

func TestWildcardNotShadowedByLiteral405(t *testing.T) {
	app := newRoutesApp(router.CollectRoutes(
		router.Get("/users/{id}", "Routes.Show"),
		router.Post("/users/search", "Routes.Create"),
	))

	rec := app.TestGet("/users/search")
	if rec.Code != http.StatusOK {
		t.Fatalf("GET /users/search should match GET /users/{id}: got %d, want 200", rec.Code)
	}
	if body := rec.Body.String(); body != `{"id":"search"}` {
		t.Fatalf("GET /users/search body: got %s", body)
	}
}

func TestAnyRouteHandlesAllMethods(t *testing.T) {
	app := newRoutesApp(router.CollectRoutes(
		router.Any("/any", "Routes.Hello"),
	))

	for _, method := range []string{http.MethodGet, http.MethodPost, http.MethodDelete} {
		rec := app.TestRequest(method, "/any", nil)
		if rec.Code != http.StatusOK {
			t.Fatalf("%s /any: got %d, want 200", method, rec.Code)
		}
	}
}

func TestAnyRouteNotShadowedBy405(t *testing.T) {
	app := newRoutesApp(router.CollectRoutes(
		router.Any("/mixed", "Routes.Hello"),
		router.Get("/mixed", "Routes.Show"),
	))

	rec := app.TestPost("/mixed", nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("POST /mixed should fall through to the ANY route: got %d, want 200", rec.Code)
	}
}

func TestStarMethodActsAsAny(t *testing.T) {
	app := newRoutesApp(router.MethodRoute("*", "/star", "Routes.Hello"))

	for _, method := range []string{http.MethodGet, http.MethodPost} {
		rec := app.TestRequest(method, "/star", nil)
		if rec.Code != http.StatusOK {
			t.Fatalf("%s /star: got %d, want 200", method, rec.Code)
		}
	}
}

func TestUserCatchAllReceivesUnmatched(t *testing.T) {
	app := newRoutesApp(router.CollectRoutes(
		router.Any("/", "Routes.Hello"),
		router.Get("/hello", "Routes.Show"),
	))

	rec := app.TestPost("/completely/unknown", nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("POST /completely/unknown should hit the user catch-all: got %d, want 200", rec.Code)
	}
}

func TestHeadServedByGet(t *testing.T) {
	app := newRoutesApp(router.CollectRoutes(
		router.Get("/hello", "Routes.Hello"),
	))

	rec := app.TestRequest(http.MethodHead, "/hello", nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("HEAD /hello: got %d, want 200", rec.Code)
	}
}
