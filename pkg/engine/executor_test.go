package engine

import (
	"log/slog"
	"testing"

	"github.com/ntnn/tensile"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// outcomeNode reports a fixed NeedsExecution result and counts executions.
type outcomeNode struct {
	name     string
	needs    bool
	executed int
}

func (n *outcomeNode) Identity() tensile.Identity {
	return tensile.AsIdentity("test", "name", n.name)
}

func (n *outcomeNode) NeedsExecution(_ tensile.Wire) (bool, tensile.Diff, error) {
	return n.needs, nil, nil
}

func (n *outcomeNode) Execute(_ tensile.Wire) (tensile.Diff, error) {
	n.executed++
	return nil, nil //nolint:nilnil // nil Diff is valid
}

func TestExecutor_Outcomes(t *testing.T) {
	t.Parallel()

	disabled := tensile.Cond(func(tensile.Wire) (bool, error) { return false, nil })

	cases := map[string]struct {
		node        *tensile.Node
		noop        bool
		wantOutcome Outcome
		wantChanged bool
	}{
		"disabled by condition": {
			node:        tensile.When(disabled, &outcomeNode{name: "n", needs: true}),
			wantOutcome: OutcomeSkipped,
		},
		"satisfied": {
			node:        tensile.NewNode(&outcomeNode{name: "n"}),
			wantOutcome: OutcomeSatisfied,
		},
		"executed": {
			node:        tensile.NewNode(&outcomeNode{name: "n", needs: true}),
			wantOutcome: OutcomeExecuted,
			wantChanged: true,
		},
		"noop": {
			node:        tensile.NewNode(&outcomeNode{name: "n", needs: true}),
			noop:        true,
			wantOutcome: OutcomeExecuted,
			wantChanged: true,
		},
		"handler satisfied": {
			node:        tensile.NewNode(tensile.NewHandler(&outcomeNode{name: "n"})),
			wantOutcome: OutcomeNotifiedSatisfied,
		},
		"handler executed": {
			node:        tensile.NewNode(tensile.NewHandler(&outcomeNode{name: "n", needs: true})),
			wantOutcome: OutcomeHandlerExecuted,
			wantChanged: true,
		},
		"handler noop": {
			node:        tensile.NewNode(tensile.NewHandler(&outcomeNode{name: "n", needs: true})),
			noop:        true,
			wantOutcome: OutcomeHandlerExecuted,
			wantChanged: true,
		},
		"handler disabled by condition": {
			node:        tensile.When(disabled, tensile.NewHandler(&outcomeNode{name: "n", needs: true})),
			wantOutcome: OutcomeSkipped,
		},
	}

	for title, cas := range cases {
		t.Run(title, func(t *testing.T) {
			t.Parallel()

			var (
				doneCalled bool
				changed    bool
			)
			executor := &Executor{
				Logger: slog.Default(),
				Noop:   cas.noop,
				Node:   cas.node,
				Wire:   &tensile.DefaultWire{},
				Store:  func(tensile.Identity, any) error { return nil },
				Done: func(c bool) {
					doneCalled = true
					changed = c
				},
			}

			ns, err := executor.Run()
			require.NoError(t, err)
			assert.Equal(t, cas.wantOutcome, ns.Outcome)
			assert.True(t, doneCalled, "Done must be called")
			assert.Equal(t, cas.wantChanged, changed)
		})
	}
}

func TestExecutor_NoopSkipsExecute(t *testing.T) {
	t.Parallel()

	wrapped := &outcomeNode{name: "n", needs: true}
	executor := &Executor{
		Logger: slog.Default(),
		Noop:   true,
		Node:   tensile.NewNode(wrapped),
		Wire:   &tensile.DefaultWire{},
		Store:  func(tensile.Identity, any) error { return nil },
		Done:   func(bool) {},
	}

	ns, err := executor.Run()
	require.NoError(t, err)
	assert.Equal(t, OutcomeExecuted, ns.Outcome)
	assert.Zero(t, wrapped.executed, "noop must not call Execute")
}
