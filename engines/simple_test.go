package engines

import (
	"context"
	"testing"

	"github.com/ntnn/tensile/nodes"
	"github.com/stretchr/testify/require"
	"golang.org/x/exp/slog"
)

func TestSimple_Run(t *testing.T) {
	n1 := &nodes.Log{
		Message: "node 1",
	}

	n2 := &nodes.Log{
		Message: "node 2",
	}

	simple, err := NewSimple(slog.Default())
	require.Nil(t, err)

	require.Nil(t, simple.Queue.Add(n1, n2))

	require.Nil(t, simple.Run(context.Background()))
}
