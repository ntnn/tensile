package engines

import (
	"context"
	"testing"

	"github.com/ntnn/gorrect"
)

func TestSimple_Add(t *testing.T) {
	simple := NewSimple()

	f, err := gorrect.NewFile("/a/target")
	if err != nil {
		t.Errorf("error creating test element: %v", err)
		return
	}

	if err := simple.Add(f); err != nil {
		t.Errorf("error adding test element: %v", err)
		return
	}

	err = simple.Add(f)
	if err != ErrSameIdentityAlreadyRegistered {
		t.Errorf("unexpected error adding the same element again: %v", err)
	}
}

func TestSimple_Run(t *testing.T) {
	f, err := gorrect.NewFile("/a/target")
	if err != nil {
		t.Errorf("error creating test file element: %v", err)
		return
	}

	d, err := gorrect.NewDir("/a")
	if err != nil {
		t.Errorf("error creating test dir element: %v", err)
		return
	}

	simple := NewSimple()
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
