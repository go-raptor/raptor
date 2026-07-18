package errs_test

import (
	"errors"
	"net/http"
	"testing"

	"github.com/go-raptor/raptor/v4/errs"
)

func TestWithCauseDoesNotMutateReceiver(t *testing.T) {
	cause := errors.New("db connection refused")

	wrapped := errs.ErrNotFound.WithCause(cause)

	if got := errs.ErrNotFound.Unwrap(); got != nil {
		t.Fatalf("WithCause mutated the shared sentinel: cause = %v", got)
	}
	if wrapped.Unwrap() != cause {
		t.Fatalf("wrapped error lost its cause: got %v, want %v", wrapped.Unwrap(), cause)
	}
	if !errors.Is(wrapped, errs.ErrNotFound) {
		t.Fatal("wrapped error no longer matches its sentinel via errors.Is")
	}
	if wrapped.Code != http.StatusNotFound || wrapped.Message != "Not Found" {
		t.Fatalf("wrapped error lost fields: code=%d message=%q", wrapped.Code, wrapped.Message)
	}
}
