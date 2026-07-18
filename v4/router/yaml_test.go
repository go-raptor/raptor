package router_test

import (
	"fmt"
	"strings"
	"testing"

	"github.com/go-raptor/raptor/v4/router"
)

func routeSet(routes router.Routes) map[string]string {
	set := make(map[string]string, len(routes))
	for _, r := range routes {
		set[r.Method+" "+r.Path] = r.Controller + "." + r.Action
	}
	return set
}

func routeSignature(routes router.Routes) string {
	var b strings.Builder
	for _, r := range routes {
		fmt.Fprintf(&b, "%s %s->%s.%s;", r.Method, r.Path, r.Controller, r.Action)
	}
	return b.String()
}

func TestParseRoutesYAMLValid(t *testing.T) {
	doc := []byte(`
routes:
  /api/v1:
    /hello:
      GET: Hello.Greet
      POST: Hello.AddGreetings
    /users/{id}: Users.Show
`)
	routes, err := router.ParseRoutesYAML(doc)
	if err != nil {
		t.Fatalf("ParseRoutesYAML: %v", err)
	}
	want := map[string]string{
		"GET /api/v1/hello":      "HelloController.Greet",
		"POST /api/v1/hello":     "HelloController.AddGreetings",
		"ANY /api/v1/users/{id}": "UsersController.Show",
	}
	got := routeSet(routes)
	if len(got) != len(want) {
		t.Fatalf("got %v, want %v", got, want)
	}
	for k, v := range want {
		if got[k] != v {
			t.Fatalf("route %q: got %q, want %q (all: %v)", k, got[k], v, got)
		}
	}
}

func TestParseRoutesYAMLDeterministicOrder(t *testing.T) {
	doc := []byte(`
routes:
  /a: A.One
  /b: B.One
  /c: C.One
  /d: D.One
  /e: E.One
  /f: F.One
`)
	first, err := router.ParseRoutesYAML(doc)
	if err != nil {
		t.Fatalf("ParseRoutesYAML: %v", err)
	}
	for i := 0; i < 20; i++ {
		next, err := router.ParseRoutesYAML(doc)
		if err != nil {
			t.Fatalf("ParseRoutesYAML: %v", err)
		}
		if routeSignature(next) != routeSignature(first) {
			t.Fatalf("route order is not deterministic:\n%s\nvs\n%s", routeSignature(first), routeSignature(next))
		}
	}
}

func TestParseRoutesYAMLRejectsNonStringMethodValue(t *testing.T) {
	doc := []byte(`
routes:
  /hello:
    GET: [Hello.Greet]
`)
	_, err := router.ParseRoutesYAML(doc)
	if err == nil {
		t.Fatal("a non-string handler for a method must be an error, not silently dropped")
	}
	if !strings.Contains(err.Error(), "GET") {
		t.Fatalf("error should name the offending method: %v", err)
	}
}

func TestParseRoutesYAMLRejectsMethodNamedKeyWithMap(t *testing.T) {
	doc := []byte(`
routes:
  get:
    /sub:
      GET: X.Y
`)
	_, err := router.ParseRoutesYAML(doc)
	if err == nil {
		t.Fatal("a method-named key with a nested map must be an error, not a silently dropped subtree")
	}
}

func TestParseRoutesYAMLRejectsInvalidPathValue(t *testing.T) {
	doc := []byte(`
routes:
  /foo: 123
`)
	_, err := router.ParseRoutesYAML(doc)
	if err == nil {
		t.Fatal("an invalid path value must be an error, not silently dropped")
	}
}

func TestParseRoutesYAMLSlashPrefixedMethodLikePath(t *testing.T) {
	doc := []byte(`
routes:
  /get: Downloads.Fetch
`)
	routes, err := router.ParseRoutesYAML(doc)
	if err != nil {
		t.Fatalf("ParseRoutesYAML: %v", err)
	}
	got := routeSet(routes)
	if got["ANY /get"] != "DownloadsController.Fetch" {
		t.Fatalf("a slash-prefixed path named like a method must stay a path: %v", got)
	}
}
