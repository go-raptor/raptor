package core_test

import (
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-raptor/raptor/v4/config"
	"github.com/go-raptor/raptor/v4/core"
)

func xffRequest(remoteAddr, xff string) *http.Request {
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.RemoteAddr = remoteAddr
	if xff != "" {
		req.Header.Set("X-Forwarded-For", xff)
	}
	return req
}

func TestXFFIgnoredWhenDirectPeerUntrusted(t *testing.T) {
	extract := core.ExtractIPFromXFFHeader()
	req := xffRequest("203.0.113.9:4444", "1.2.3.4")
	if got := extract(req); got != "203.0.113.9" {
		t.Fatalf("spoofable XFF from an untrusted peer must be ignored: got %q", got)
	}
}

func TestXFFWalksToFirstUntrustedHop(t *testing.T) {
	extract := core.ExtractIPFromXFFHeader()
	req := xffRequest("10.0.0.9:4444", "6.6.6.6, 10.0.0.1")
	if got := extract(req); got != "6.6.6.6" {
		t.Fatalf("got %q, want first untrusted hop 6.6.6.6", got)
	}
}

func TestXFFGarbageEntryFallsBackToDirect(t *testing.T) {
	extract := core.ExtractIPFromXFFHeader()
	req := xffRequest("10.0.0.9:4444", "evil-garbage, 10.0.0.1")
	if got := extract(req); got != "10.0.0.9" {
		t.Fatalf("unparsable XFF entry must fall back to the direct peer: got %q", got)
	}
}

func TestXFFAllTrustedFallbackIsNormalized(t *testing.T) {
	extract := core.ExtractIPFromXFFHeader()
	req := xffRequest("10.0.0.9:4444", "[fd00::1], 10.0.0.2")
	if got := extract(req); got != "fd00::1" {
		t.Fatalf("all-trusted fallback must return a validated, normalized IP: got %q", got)
	}
}

func TestRealIPHonoredOnlyFromTrustedPeer(t *testing.T) {
	extract := core.ExtractIPFromRealIPHeader()

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.RemoteAddr = "127.0.0.1:4444"
	req.Header.Set("X-Real-Ip", "9.9.9.9")
	if got := extract(req); got != "9.9.9.9" {
		t.Fatalf("trusted peer: got %q, want 9.9.9.9", got)
	}

	req.RemoteAddr = "203.0.113.9:4444"
	if got := extract(req); got != "203.0.113.9" {
		t.Fatalf("untrusted peer: got %q, want the direct address", got)
	}
}

func TestTrustedProxiesCustomCIDR(t *testing.T) {
	trusted, err := core.TrustedProxies([]string{"203.0.113.0/24"})
	if err != nil {
		t.Fatalf("TrustedProxies: %v", err)
	}
	extract := core.ExtractIPFromXFFHeader(trusted)

	req := xffRequest("203.0.113.9:4444", "9.9.9.9")
	if got := extract(req); got != "9.9.9.9" {
		t.Fatalf("peer inside the trusted CIDR: got %q, want 9.9.9.9", got)
	}

	req = xffRequest("10.1.2.3:4444", "9.9.9.9")
	if got := extract(req); got != "10.1.2.3" {
		t.Fatalf("custom trusted list replaces the default private-range trust: got %q", got)
	}
}

func TestTrustedProxiesRejectsInvalidEntry(t *testing.T) {
	if _, err := core.TrustedProxies([]string{"not-a-cidr"}); err == nil {
		t.Fatal("invalid trusted proxy entry must be rejected")
	}
}

func TestNewCoreWiresTrustedProxiesFromConfig(t *testing.T) {
	resources := core.NewResources()
	resources.SetLogHandler(slog.NewTextHandler(io.Discard, nil))
	cfg := config.NewConfigDefaults()
	cfg.ServerConfig.IPExtractor = "x-forwarded-for"
	cfg.ServerConfig.TrustedProxies = []string{"203.0.113.0/24"}
	resources.SetConfig(cfg)
	c := core.NewCore(resources)

	ctx := core.NewContext(c, xffRequest("203.0.113.9:4444", "9.9.9.9"), httptest.NewRecorder())
	if got := ctx.RealIP(); got != "9.9.9.9" {
		t.Fatalf("trusted_proxies config not wired into the extractor: got %q", got)
	}
}

func TestNewCorePanicsOnInvalidTrustedProxyConfig(t *testing.T) {
	resources := core.NewResources()
	resources.SetLogHandler(slog.NewTextHandler(io.Discard, nil))
	cfg := config.NewConfigDefaults()
	cfg.ServerConfig.TrustedProxies = []string{"not-a-cidr"}
	resources.SetConfig(cfg)

	defer func() {
		if recover() == nil {
			t.Fatal("invalid trusted_proxies must fail startup, not silently degrade")
		}
	}()
	core.NewCore(resources)
}
