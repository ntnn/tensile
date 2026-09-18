package tensile

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// condNode counts executions and reports a fixed output.
type condNode struct {
	executed int
	deps     []Identity
}

func (n *condNode) Identity() Identity {
	return AsIdentity("cond", "name", "node")
}

func (n *condNode) DependsOn() ([]Identity, error) {
	return n.deps, nil
}

func (n *condNode) NeedsExecution(_ Wire) (bool, error) {
	return true, nil
}

func (n *condNode) Execute(_ Wire) error {
	n.executed++
	return nil
}

func (n *condNode) Report(_ Wire) (any, error) {
	return "output", nil
}

func TestWhen_DisabledSkipsExecutionButReports(t *testing.T) {
	t.Parallel()

	wire := &DefaultWire{}
	wrapped := &condNode{}
	node := When(
		Cond(func(Wire) (bool, error) {
			return false, nil
		}),
		wrapped,
	)

	needs, err := node.NeedsExecution(wire)
	require.NoError(t, err)
	assert.False(t, needs, "disabled node must not need execution")

	require.NoError(t, node.Execute(wire))
	assert.Zero(t, wrapped.executed, "disabled node must not execute")

	output, ok, err := node.Report(wire)
	require.NoError(t, err)
	require.True(t, ok, "disabled node must still report")
	assert.Equal(t, "output", output)
}

func TestWhen_EnabledPassesThrough(t *testing.T) {
	t.Parallel()

	wire := &DefaultWire{}
	wrapped := &condNode{}
	node := When(
		Cond(func(Wire) (bool, error) {
			return true, nil
		}),
		wrapped,
	)

	needs, err := node.NeedsExecution(wire)
	require.NoError(t, err)
	assert.True(t, needs)

	require.NoError(t, node.Execute(wire))
	assert.Equal(t, 1, wrapped.executed)

	output, ok, err := node.Report(wire)
	require.NoError(t, err)
	require.True(t, ok)
	assert.Equal(t, "output", output)
}

func TestWhen_ConditionErrorPropagates(t *testing.T) {
	t.Parallel()

	wire := &DefaultWire{}
	errCond := errors.New("cond failed")
	node := When(
		Cond(func(Wire) (bool, error) {
			return false, errCond
		}),
		&condNode{},
	)

	_, err := node.NeedsExecution(wire)
	require.ErrorIs(t, err, errCond)
	require.ErrorIs(t, node.Execute(wire), errCond)
	require.ErrorIs(t, node.Validate(wire), errCond)
}

func TestWhen_AppendsReads(t *testing.T) {
	t.Parallel()

	own := AsIdentity("dep", "name", "own")
	extra := AsIdentity("dep", "name", "extra")
	node := When(
		Cond(
			func(Wire) (bool, error) {
				return true, nil
			},
			extra,
		),
		&condNode{deps: []Identity{own}},
	)

	deps, err := node.DependsOn()
	require.NoError(t, err)
	assert.Equal(t, []Identity{own, extra}, deps)
}
