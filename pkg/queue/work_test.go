package queue_test

import (
	"context"
	"testing"
	"time"

	"github.com/ntnn/tensile"
	"github.com/ntnn/tensile/pkg/queue"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type testNode struct {
	Name     string
	Provide  []tensile.Identity
	DependOn []tensile.Identity
}

func (n testNode) Identity() tensile.Identity {
	return tensile.AsIdentity("test", "name", n.Name)
}

func (n testNode) Provides() ([]tensile.Identity, error) {
	return n.Provide, nil
}

func (n testNode) DependsOn() ([]tensile.Identity, error) {
	return n.DependOn, nil
}

func buildWork(t *testing.T, nodes ...tensile.Identifier) *queue.Work {
	t.Helper()
	q := queue.New()
	require.NoError(t, q.Enqueue(nodes...))
	work, err := q.Build()
	require.NoError(t, err)
	return work
}

// buildNotifiedWork builds a Work with a single notifier node and a
// handler notified by it, returning the work and the notifier's identity.
func buildNotifiedWork(t *testing.T) (*queue.Work, tensile.Identity, *tensile.Handler) {
	t.Helper()

	node := testNode{Name: "notifier"}
	handler := tensile.NewHandler(testNode{Name: "handler"})

	q := queue.New()
	require.NoError(t, q.Enqueue(node, handler))
	require.NoError(t, q.NotifiedBy(handler, node))

	work, err := q.Build()
	require.NoError(t, err)

	return work, nodeIdentity(t, node), handler
}

func nodeIdentity(t *testing.T, input tensile.Identifier) tensile.Identity {
	t.Helper()
	return tensile.NewNode(input).Identity()
}

func TestWork_ChanYieldsAllIndependentNodesAndCloses(t *testing.T) {
	t.Parallel()

	a := testNode{Name: "a"}
	b := testNode{Name: "b"}
	c := testNode{Name: "c"}
	work := buildWork(t, a, b, c)

	want := map[tensile.Identity]bool{
		nodeIdentity(t, a): true,
		nodeIdentity(t, b): true,
		nodeIdentity(t, c): true,
	}

	got := map[tensile.Identity]bool{}
	for item := range work.Chan(t.Context()) {
		require.NoError(t, item.Err)
		got[item.Node.Identity()] = true
	}
	assert.Equal(t, want, got, "every node must be yielded exactly once")
}

func TestWork_ChanBlocksDependentUntilProviderDone(t *testing.T) {
	t.Parallel()

	ref := tensile.AsIdentity("testres", "name", "dep")
	provider := testNode{Name: "provider", Provide: []tensile.Identity{ref}}
	depender := testNode{Name: "depender", DependOn: []tensile.Identity{ref}}
	work := buildWork(t, provider, depender)

	items := work.Chan(t.Context())

	first := <-items
	require.NoError(t, first.Err)
	require.Equal(t, nodeIdentity(t, provider), first.Node.Identity())

	select {
	case item := <-items:
		t.Fatalf("depender yielded before provider was done: %+v", item)
	case <-time.After(50 * time.Millisecond):
	}

	work.MarkDone(first.Node, true)

	second := <-items
	require.NoError(t, second.Err)
	assert.Equal(t, nodeIdentity(t, depender), second.Node.Identity())

	_, open := <-items
	assert.False(t, open, "channel must be closed after all nodes were yielded")
}

func TestWork_ChanYieldsContextErrorWhileWaiting(t *testing.T) {
	t.Parallel()

	ref := tensile.AsIdentity("testres", "name", "dep")
	provider := testNode{Name: "provider", Provide: []tensile.Identity{ref}}
	depender := testNode{Name: "depender", DependOn: []tensile.Identity{ref}}
	work := buildWork(t, provider, depender)

	ctx, cancel := context.WithCancel(t.Context())
	items := work.Chan(ctx)

	first := <-items
	require.NoError(t, first.Err)

	// Feeder is now waiting for MarkDone; cancel must wake it.
	cancel()

	item := <-items
	require.ErrorIs(t, item.Err, context.Canceled)
	assert.Nil(t, item.Node, "error item must not carry a node")

	_, open := <-items
	assert.False(t, open, "channel must be closed after the error")
}

func TestWork_ChanSkipsHandlerWithoutExecutedNotifier(t *testing.T) {
	t.Parallel()

	work, notifierIdentity, _ := buildNotifiedWork(t)

	items := work.Chan(t.Context())

	first := <-items
	require.NoError(t, first.Err)
	require.Equal(t, notifierIdentity, first.Node.Identity())
	work.MarkDone(first.Node, false)

	item, open := <-items
	assert.False(t, open, "handler without executed notifier must be skipped: %+v", item)
}

func TestWork_ChanYieldsHandlerAfterNotifierExecuted(t *testing.T) {
	t.Parallel()

	work, notifierIdentity, handler := buildNotifiedWork(t)

	items := work.Chan(t.Context())

	first := <-items
	require.NoError(t, first.Err)
	require.Equal(t, notifierIdentity, first.Node.Identity())
	work.MarkDone(first.Node, true)

	second := <-items
	require.NoError(t, second.Err)
	assert.Equal(t, handler.Identity(), second.Node.Identity())
}

func TestWork_ChanBlocksManualDependencyUntilDone(t *testing.T) {
	t.Parallel()

	first := testNode{Name: "first"}
	second := testNode{Name: "second"}

	q := queue.New()
	require.NoError(t, q.Enqueue(first, second))
	require.NoError(t, q.Depends(second, first))
	work, err := q.Build()
	require.NoError(t, err)

	items := work.Chan(t.Context())

	item := <-items
	require.NoError(t, item.Err)
	require.Equal(t, nodeIdentity(t, first), item.Node.Identity())

	// first is yielded but not done; second must not be yielded yet
	select {
	case got := <-items:
		t.Fatalf("depender yielded before its manual dependency was done: %+v", got)
	case <-time.After(50 * time.Millisecond):
	}

	work.MarkDone(item.Node, true)

	item = <-items
	require.NoError(t, item.Err)
	assert.Equal(t, nodeIdentity(t, second), item.Node.Identity())
}
