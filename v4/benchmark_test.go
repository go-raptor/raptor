package raptor_test

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/go-raptor/raptor/v4"
	"github.com/go-raptor/raptor/v4/router"
)

type BenchController struct {
	raptor.Controller
}

func (c *BenchController) Hello(ctx *raptor.Context) error {
	return ctx.Data(map[string]string{"message": "hello"})
}

func (c *BenchController) Show(ctx *raptor.Context) error {
	return ctx.Data(map[string]string{"id": ctx.Param("id")})
}

func (c *BenchController) Create(ctx *raptor.Context) error {
	var request struct {
		Name string `json:"name"`
	}
	if err := ctx.Bind(&request); err != nil {
		return err
	}
	return ctx.Data(request, http.StatusCreated)
}

func newBenchApp() *raptor.Raptor {
	return raptor.NewTestApp(
		&raptor.Components{Controllers: raptor.Controllers{&BenchController{}}},
		router.CollectRoutes(
			router.Get("/hello", "Bench.Hello"),
			router.Get("/things/{id}", "Bench.Show"),
			router.Post("/things", "Bench.Create"),
		),
	)
}

func BenchmarkServeJSON(b *testing.B) {
	app := newBenchApp()
	req := httptest.NewRequest(http.MethodGet, "/hello", nil)
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		rec := httptest.NewRecorder()
		app.ServeHTTP(rec, req)
	}
}

func BenchmarkServeParam(b *testing.B) {
	app := newBenchApp()
	req := httptest.NewRequest(http.MethodGet, "/things/42", nil)
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		rec := httptest.NewRecorder()
		app.ServeHTTP(rec, req)
	}
}

func BenchmarkServeBind(b *testing.B) {
	app := newBenchApp()
	payload := `{"name":"raptor"}`
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		req := httptest.NewRequest(http.MethodPost, "/things", strings.NewReader(payload))
		req.Header.Set("Content-Type", "application/json")
		rec := httptest.NewRecorder()
		app.ServeHTTP(rec, req)
	}
}

func BenchmarkServeNotFound(b *testing.B) {
	app := newBenchApp()
	req := httptest.NewRequest(http.MethodGet, "/nope", nil)
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		rec := httptest.NewRecorder()
		app.ServeHTTP(rec, req)
	}
}
