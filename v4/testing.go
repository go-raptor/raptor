package raptor

import (
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"

	"github.com/go-raptor/raptor/v4/config"
	"github.com/go-raptor/raptor/v4/core"
	"github.com/go-raptor/raptor/v4/router"
)

type TestRequestOption func(*http.Request)

func NewTestApp(components *core.Components, routes router.Routes, opts ...RaptorOption) *Raptor {
	return New(components, routes, append([]RaptorOption{withTestDefaults()}, opts...)...)
}

func withTestDefaults() RaptorOption {
	return func(r *Raptor) {
		r.testMode = true
		// Suppress log output during test app startup
		r.resources.SetLogLevel("error")
		r.configOverride = &config.Config{
			GeneralConfig: config.GeneralConfig{
				LogLevel: "error",
			},
		}
	}
}

func NewTestResources() *core.Resources {
	resources := core.NewResources()

	cfg := config.NewConfigDefaults()
	cfg.GeneralConfig.LogLevel = "error"
	resources.SetConfig(cfg)

	return resources
}

func (r *Raptor) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	r.Router.Mux.ServeHTTP(w, req)
}

func (r *Raptor) TestRequest(method, path string, body io.Reader, opts ...TestRequestOption) *httptest.ResponseRecorder {
	req := httptest.NewRequest(method, path, body)
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	for _, opt := range opts {
		opt(req)
	}
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)
	return rec
}

func WithHeader(key, value string) TestRequestOption {
	return func(req *http.Request) {
		req.Header.Set(key, value)
	}
}

// WithRemoteAddr sets the client address. httptest gives every request
// 192.0.2.1:1234, so without it a whole suite shares one ctx.RealIP() and
// one bucket in any per-IP middleware. addr may omit the port, and an IPv6
// address may be bracketed or bare.
func WithRemoteAddr(addr string) TestRequestOption {
	if _, _, err := net.SplitHostPort(addr); err != nil {
		addr = net.JoinHostPort(strings.Trim(addr, "[]"), "1234")
	}
	return func(req *http.Request) {
		req.RemoteAddr = addr
	}
}

func (r *Raptor) TestGet(path string, opts ...TestRequestOption) *httptest.ResponseRecorder {
	return r.TestRequest(http.MethodGet, path, nil, opts...)
}

func (r *Raptor) TestPost(path string, body io.Reader, opts ...TestRequestOption) *httptest.ResponseRecorder {
	return r.TestRequest(http.MethodPost, path, body, opts...)
}

func (r *Raptor) TestPut(path string, body io.Reader, opts ...TestRequestOption) *httptest.ResponseRecorder {
	return r.TestRequest(http.MethodPut, path, body, opts...)
}

func (r *Raptor) TestPatch(path string, body io.Reader, opts ...TestRequestOption) *httptest.ResponseRecorder {
	return r.TestRequest(http.MethodPatch, path, body, opts...)
}

func (r *Raptor) TestDelete(path string, opts ...TestRequestOption) *httptest.ResponseRecorder {
	return r.TestRequest(http.MethodDelete, path, nil, opts...)
}
