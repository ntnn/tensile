package engines

import (
	"context"
	"testing"

	"github.com/ntnn/tensile/nodes"
	"github.com/stretchr/testify/require"
	"golang.org/x/exp/slog"
)

func TestSequential_Run(t *testing.T) {
	n1 := &nodes.Log{
		Message: "node 1",
	}

	n2 := &nodes.Log{
		Message: "node 2",
	}

	seq, err := NewSequential(slog.Default())
	require.Nil(t, err)

	require.Nil(t, seq.Queue.Add(n1, n2))

	require.Nil(t, seq.Run(context.Background()))
}
