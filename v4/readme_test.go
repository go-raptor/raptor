package raptor_test

import (
	"os"
	"regexp"
	"testing"

	"github.com/go-raptor/raptor/v4"
)

// currentVersion matches the two places the README names the current
// release, not feature floors such as "(v4.6.0+)".
var currentVersion = regexp.MustCompile(`Raptor (v4\.\d+\.\d+) is running|current release is \*\*(v4\.\d+\.\d+)\*\*`)

func TestReadmeNamesCurrentVersion(t *testing.T) {
	// The README lives at the repository root, outside the module, so it
	// is absent when the module is tested from the module cache.
	b, err := os.ReadFile("../README.md")
	if os.IsNotExist(err) {
		t.Skip("README.md is outside the module")
	}
	if err != nil {
		t.Fatal(err)
	}
	matches := currentVersion.FindAllSubmatch(b, -1)
	if len(matches) < 2 {
		t.Fatalf("README.md must name the current release in the Quickstart banner and in Project status; found %d", len(matches))
	}
	for _, m := range matches {
		if got := string(m[1]) + string(m[2]); got != raptor.Version {
			t.Errorf("README.md says %s, raptor.Version is %s", got, raptor.Version)
		}
	}
}
