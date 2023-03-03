package engines

import (
	"context"
	"testing"

	"github.com/ntnn/gorrect"
	"golang.org/x/exp/slog"
)

func TestSimple_Run(t *testing.T) {
	f := &gorrect.File{
		Target: "/a/target",
	}

	d := &gorrect.Dir{
		Target: "/a",
	}

	simple := NewSimple(slog.Default())
	if err := simple.Queue.Add(f, d); err != nil {
		t.Errorf("error adding test element: %v", err)
		return
	}

	if err := simple.Run(context.Background()); err != nil {
		t.Errorf("error in execution: %v", err)
	}
}
