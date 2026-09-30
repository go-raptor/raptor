package raptor_test

import (
	"bytes"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
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

type FileBenchController struct {
	raptor.Controller
	dir string
}

func (c *FileBenchController) Serve(ctx *raptor.Context) error {
	return ctx.FileFromDir(c.dir, "big.bin")
}

// BenchmarkServeFile fetches an 8 MB file over a real socket, the only
// setup where net/http can use sendfile, and only if every writer in the
// chain lets io.Copy reach net/http's ReadFrom.
func BenchmarkServeFile(b *testing.B) {
	dir := b.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "big.bin"), bytes.Repeat([]byte("x"), 8<<20), 0o644); err != nil {
		b.Fatal(err)
	}
	app := raptor.NewTestApp(
		&raptor.Components{Controllers: raptor.Controllers{&FileBenchController{dir: dir}}},
		router.Get("/big.bin", "FileBench.Serve"),
	)
	srv := httptest.NewServer(app)
	defer srv.Close()
	client := srv.Client()

	b.SetBytes(8 << 20)
	b.ReportAllocs()
	for b.Loop() {
		resp, err := client.Get(srv.URL + "/big.bin")
		if err != nil {
			b.Fatal(err)
		}
		if _, err := io.Copy(io.Discard, resp.Body); err != nil {
			b.Fatal(err)
		}
		resp.Body.Close()
	}
}
