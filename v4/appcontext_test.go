package raptor

import (
	"context"
	"errors"
	"testing"
)

// A test checks that a service's background work stops by cancelling its app
// context, as shutdown does once requests have drained.
func TestCancelAppContext(t *testing.T) {
	res := NewTestResources()
	ctx := res.AppContext()
	if ctx.Err() != nil {
		t.Fatal("the app context is cancelled before CancelAppContext")
	}

	CancelAppContext(res)
	if !errors.Is(ctx.Err(), context.Canceled) {
		t.Errorf("after CancelAppContext the app context's error is %v, want context.Canceled", ctx.Err())
	}
	CancelAppContext(res) // a second call is harmless, as with any cancel func
}

// A service holds the resources Raptor injected, so it sees the cancellation.
func TestCancelAppContextReachesAService(t *testing.T) {
	res := NewTestResources()
	s := &Service{}
	if err := s.Init(res); err != nil {
		t.Fatal(err)
	}

	CancelAppContext(res)
	select {
	case <-s.AppContext().Done():
	default:
		t.Error("the service's AppContext is still live after CancelAppContext")
	}
}

// Resources built by hand have no app context to cancel: nothing happens.
func TestCancelAppContextWithoutOne(t *testing.T) {
	CancelAppContext(&Resources{})
}
