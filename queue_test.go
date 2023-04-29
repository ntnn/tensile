package tensile

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
)

type testNode struct {
	Name string
}

func (t testNode) Shape() Shape {
	return Noop
}

func (t testNode) Identifier() string {
	return t.Name
}

var _ IsCollisioner = (*&testNodeCollisioner{})

type testNodeCollisioner struct {
	Name string
}

func (t testNodeCollisioner) Shape() Shape {
	return Noop
}

func (t testNodeCollisioner) Identifier() string {
	return t.Name
}

func (t testNodeCollisioner) IsCollision(other Node) error {
	return nil
}

func TestQueue_Add(t *testing.T) {
	q := NewQueue()

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

func testQueue_Channel_compare(t *testing.T, q *Queue, expected []string, expectedErr error) {
	ch, errCh := q.Channel(context.Background())

	ret := []string{}
	for node := range ch {
		ret = append(ret, node.String())
	}
	require.Equal(t, expectedErr, <-errCh)
	require.Equal(t, expected, ret)
}

func TestQueue_Channel(t *testing.T) {
	q := NewQueue()

	require.Nil(t, q.Add(
		&NodeWrapper{
			Node: &testNode{Name: "first"},
			After: []string{
				"noop[second]",
			},
		},
	))

	require.Nil(t, q.Add(
		&NodeWrapper{
			Node: &testNode{Name: "second"},
			Before: []string{
				"noop[first]",
			},
		},
	))

	testQueue_Channel_compare(t, q,
		[]string{
			"noop[second]",
			"noop[first]",
		},
		nil,
	)

	require.Nil(t, q.Add(
		&NodeWrapper{
			Node: &testNode{Name: "third"},
			After: []string{
				"noop[second]",
			},
		},
	))

	testQueue_Channel_compare(t, q,
		[]string{
			"noop[second]",
			"noop[first]",
			"noop[third]",
		},
		nil,
	)

	require.Nil(t, q.Add(
		&NodeWrapper{
			Node: &testNode{Name: "fourth"},
			Before: []string{
				"noop[second]",
			},
		},
	))

	testQueue_Channel_compare(t, q,
		[]string{
			"noop[fourth]",
			"noop[second]",
			"noop[first]",
			"noop[third]",
		},
		nil,
	)

	require.Nil(t, q.Add(
		&NodeWrapper{
			Node: &testNode{Name: "fifth"},
			After: []string{
				"noop[first]",
			},
			Before: []string{
				"noop[first]",
			},
		},
	))
	testQueue_Channel_compare(t, q,
		[]string{
			"noop[fourth]",
			"noop[second]",
			"noop[third]",
		},
		ErrCyclicalDependencies,
	)
}
