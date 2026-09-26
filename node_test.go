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

func (n *condNode) NeedsExecution(_ Wire) (bool, Diff, error) {
	return true, nil, nil
}

func (n *condNode) Execute(_ Wire) (Diff, error) {
	n.executed++
	return nil, nil //nolint:nilnil // nil Diff is valid
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

	needs, _, err := node.NeedsExecution(wire)
	require.NoError(t, err)
	assert.False(t, needs, "disabled node must not need execution")

	_, err = node.Execute(wire)
	require.NoError(t, err)
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

	needs, _, err := node.NeedsExecution(wire)
	require.NoError(t, err)
	assert.True(t, needs)

	_, err = node.Execute(wire)
	require.NoError(t, err)
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

	_, _, err := node.NeedsExecution(wire)
	require.ErrorIs(t, err, errCond)
	_, err = node.Execute(wire)
	require.ErrorIs(t, err, errCond)
	require.ErrorIs(t, node.Validate(wire), errCond)
}

func TestNode_Enabled(t *testing.T) {
	t.Parallel()

	wire := &DefaultWire{}

	enabled, err := NewNode(&condNode{}).Enabled(wire)
	require.NoError(t, err)
	assert.True(t, enabled, "nil condition must be enabled")

	enabled, err = When(Cond(func(Wire) (bool, error) { return true, nil }), &condNode{}).Enabled(wire)
	require.NoError(t, err)
	assert.True(t, enabled)

	enabled, err = When(Cond(func(Wire) (bool, error) { return false, nil }), &condNode{}).Enabled(wire)
	require.NoError(t, err)
	assert.False(t, enabled)
}

func TestNode_IsHandler(t *testing.T) {
	t.Parallel()

	assert.False(t, NewNode(&condNode{}).IsHandler())

	handler := NewHandler(&condNode{})
	assert.True(t, handler.IsHandler())
	assert.True(t, NewNode(handler).IsHandler(), "NewNode must keep the handler flag")
	assert.True(t, NewHandler(handler).IsHandler(), "NewHandler must pass a handler through")
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
