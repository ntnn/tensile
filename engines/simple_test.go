package engines

import (
	"context"
	"testing"

	"github.com/ntnn/tensile/nodes"
	"golang.org/x/exp/slog"
)

func TestSimple_Run(t *testing.T) {
	n1 := &nodes.Log{
		Message: "node 1",
	}

	n2 := &nodes.Log{
		Message: "node 2",
	}

	simple := NewSimple(slog.Default())
	if err := simple.Queue.Add(n1, n2); err != nil {
		t.Errorf("error adding test element: %v", err)
		return
	}

	if err := simple.Run(context.Background()); err != nil {
		t.Errorf("error in execution: %v", err)
	}
}
