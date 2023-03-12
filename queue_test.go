package tensile

import (
	"testing"

	"github.com/ntnn/tensile/facts"
	"github.com/stretchr/testify/require"
)

type testNode struct {
	Name string
}

func (t testNode) Identity() (Shape, string) {
	return Noop, t.Name
}

type testNodeCollisioner struct {
	Name string
}

func (t testNodeCollisioner) Identity() (Shape, string) {
	return Noop, t.Name
}

func (t testNodeCollisioner) IsCollision(other Identitier) error {
	return nil
}

func TestQueue_Add(t *testing.T) {
	q := NewQueue(facts.Facts{})

	// Adding nodes
	require.Nil(t, q.Add(&testNode{Name: "Hello, world!"}))
	require.NotNil(t, q.nodes["noop[Hello, world!]"])
	require.Nil(t, q.Add(&testNode{Name: "Hello, world 2!"}))
	require.NotNil(t, q.nodes["noop[Hello, world 2!]"])
	require.Nil(t, q.Add(&testNode{Name: "Hello, world 3!"}))
	require.NotNil(t, q.nodes["noop[Hello, world 3!]"])

	// Adding multiples nodes in one call
	require.Nil(t, q.Add(
		&testNode{Name: "Hello, world 4!"},
		&testNode{Name: "Hello, world 5!"},
		&testNode{Name: "Hello, world 6!"},
	))
	require.NotNil(t, q.nodes["noop[Hello, world 4!]"])
	require.NotNil(t, q.nodes["noop[Hello, world 5!]"])
	require.NotNil(t, q.nodes["noop[Hello, world 6!]"])

	// Colliding nodes generate an error
	require.Nil(t, q.Add(&testNode{Name: "Hello, collision!"}))
	require.NotNil(t, q.nodes["noop[Hello, collision!]"])
	require.Errorf(t,
		q.Add(&testNode{Name: "Hello, collision!"}),
		"same identity already registered, collision check: %w", ErrIsCollisionerNotImplemented,
	)

	// Nodes with IsCollisioner do not cause a collision
	require.Nil(t, q.Add(&testNodeCollisioner{Name: "Hello, no collision!"}))
	require.Nil(t, q.Add(&testNodeCollisioner{Name: "Hello, no collision!"}))
	require.NotNil(t, q.nodes["noop[Hello, no collision!]"])

	// One node with IsCollisioner returning nil and one node without
	// IsCollisioner do not cause a collision
	require.Nil(t, q.Add(&testNodeCollisioner{Name: "Hello, first IsCollisioner, second no IsCollisioner!"}))
	require.Nil(t, q.Add(&testNode{Name: "Hello, first IsCollisioner, second no IsCollisioner!"}))
	require.NotNil(t, q.nodes["noop[Hello, first IsCollisioner, second no IsCollisioner!]"])

	// And the reverse
	require.Nil(t, q.Add(&testNode{Name: "Hello, first no IsCollisioner, second IsCollisioner!"}))
	require.Nil(t, q.Add(&testNodeCollisioner{Name: "Hello, first no IsCollisioner, second IsCollisioner!"}))
	require.NotNil(t, q.nodes["noop[Hello, first no IsCollisioner, second IsCollisioner!]"])
}
