package engines

import (
	"context"
	"testing"

	"github.com/ntnn/gorrect"
	"golang.org/x/exp/slog"
)

func TestSimple_Add(t *testing.T) {
	simple := NewSimple(slog.Default())

	f := &gorrect.File{
		Target: "/a/target",
	}

	if err := simple.Add(f); err != nil {
		t.Errorf("error adding test element: %v", err)
		return
	}

	if err := simple.Add(f); err != ErrSameIdentityAlreadyRegistered {
		t.Errorf("unexpected error adding the same element again: %v", err)
	}
}

func TestSimple_Run(t *testing.T) {
	f := &gorrect.File{
		Target: "/a/target",
	}

	d := &gorrect.Dir{
		Target: "/a",
	}

	simple := NewSimple(slog.Default())
	if err := simple.Add(f); err != nil {
		t.Errorf("error adding test file element: %v", err)
		return
	}

	if err := simple.Add(d); err != nil {
		t.Errorf("error adding test dir element: %v", err)
		return
	}

	if err := simple.Run(context.Background()); err != nil {
		t.Errorf("error in execution: %v", err)
	}
}
