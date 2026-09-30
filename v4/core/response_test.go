package core_test

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/go-raptor/raptor/v4/core"
)

// readFromWriter stands in for net/http's writer, whose ReadFrom uses
// sendfile, and records whether io.Copy reached it.
type readFromWriter struct {
	*httptest.ResponseRecorder
	readFromCalls int
}

func (w *readFromWriter) ReadFrom(src io.Reader) (int64, error) {
	w.readFromCalls++
	return io.Copy(w.ResponseRecorder, src)
}

// limited mirrors http.ServeContent, which copies with io.CopyN: a
// LimitedReader has no WriteTo, so io.Copy must use the destination's
// ReadFrom or fall back to a buffered copy.
func limited(s string) io.Reader {
	return io.LimitReader(strings.NewReader(s), int64(len(s)))
}

func TestResponseReadFromReachesUnderlyingWriter(t *testing.T) {
	w := &readFromWriter{ResponseRecorder: httptest.NewRecorder()}
	res := core.NewResponse(w)

	n, err := io.Copy(res, limited("hello"))
	if err != nil || n != 5 {
		t.Fatalf("io.Copy = %d, %v", n, err)
	}
	if w.readFromCalls != 1 {
		t.Fatalf("underlying ReadFrom called %d times, want 1: file responses would miss sendfile", w.readFromCalls)
	}
	if w.Body.String() != "hello" {
		t.Fatalf("body %q", w.Body.String())
	}
}

func TestResponseReadFromCommitsOK(t *testing.T) {
	rec := httptest.NewRecorder()
	res := core.NewResponse(rec)

	if _, err := res.ReadFrom(limited("hello")); err != nil {
		t.Fatal(err)
	}
	if !res.Committed || res.Status != http.StatusOK || rec.Code != http.StatusOK {
		t.Fatalf("committed=%v status=%d recorded=%d, want a committed 200", res.Committed, res.Status, rec.Code)
	}
	if res.Size != 5 {
		t.Fatalf("Size = %d, want 5", res.Size)
	}
}

func TestResponseReadFromKeepsWrittenStatus(t *testing.T) {
	rec := httptest.NewRecorder()
	res := core.NewResponse(rec)

	res.WriteHeader(http.StatusPartialContent)
	if _, err := res.ReadFrom(limited("hel")); err != nil {
		t.Fatal(err)
	}
	if res.Status != http.StatusPartialContent || rec.Code != http.StatusPartialContent {
		t.Fatalf("status=%d recorded=%d, want 206", res.Status, rec.Code)
	}
	if res.Size != 3 {
		t.Fatalf("Size = %d, want 3", res.Size)
	}
}
