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
	Provide  []tensile.NodeRef
	DependOn []tensile.NodeRef
}

func (n testNode) Provides() ([]tensile.NodeRef, error) {
	return n.Provide, nil
}

func (n testNode) DependsOn() ([]tensile.NodeRef, error) {
	return n.DependOn, nil
}

func buildWork(t *testing.T, nodes ...any) *queue.Work {
	t.Helper()
	q := queue.New()
	require.NoError(t, q.Enqueue(nodes...))
	work, err := q.Build()
	require.NoError(t, err)
	return work
}

// buildNotifiedWork builds a Work with a single notifier node and a
// handler notified by it, returning the work and the notifier's ID.
func buildNotifiedWork(t *testing.T) (*queue.Work, int64, *tensile.Handler) {
	t.Helper()

	node := testNode{Name: "notifier"}
	handler, err := tensile.NewHandler(testNode{Name: "handler"})
	require.NoError(t, err)

	q := queue.New()
	require.NoError(t, q.Enqueue(node, handler))
	require.NoError(t, q.NotifiedBy(handler, node))
	work, err := q.Build()
	require.NoError(t, err)
	return work, nodeID(t, node), handler
}

func nodeID(t *testing.T, input any) int64 {
	t.Helper()
	node, err := tensile.NewNode(input)
	require.NoError(t, err)
	return node.ID()
}

func TestWork_ChanYieldsAllIndependentNodesAndCloses(t *testing.T) {
	t.Parallel()

	a := testNode{Name: "a"}
	b := testNode{Name: "b"}
	c := testNode{Name: "c"}
	work := buildWork(t, a, b, c)

	want := map[int64]bool{
		nodeID(t, a): true,
		nodeID(t, b): true,
		nodeID(t, c): true,
	}

	got := map[int64]bool{}
	for item := range work.Chan(t.Context()) {
		require.NoError(t, item.Err)
		got[item.Node.ID()] = true
	}
	assert.Equal(t, want, got, "every node must be yielded exactly once")
}

func TestWork_ChanBlocksDependentUntilProviderDone(t *testing.T) {
	t.Parallel()

	ref := tensile.Ref("test").To("dep")
	provider := testNode{Name: "provider", Provide: []tensile.NodeRef{ref}}
	depender := testNode{Name: "depender", DependOn: []tensile.NodeRef{ref}}
	work := buildWork(t, provider, depender)

	items := work.Chan(t.Context())

	first := <-items
	require.NoError(t, first.Err)
	require.Equal(t, nodeID(t, provider), first.Node.ID())

	select {
	case item := <-items:
		t.Fatalf("depender yielded before provider was done: %+v", item)
	case <-time.After(50 * time.Millisecond):
	}

	work.MarkDone(first.Node, true)

	second := <-items
	require.NoError(t, second.Err)
	assert.Equal(t, nodeID(t, depender), second.Node.ID())

	_, open := <-items
	assert.False(t, open, "channel must be closed after all nodes were yielded")
}

func TestWork_ChanYieldsContextErrorWhileWaiting(t *testing.T) {
	t.Parallel()

	ref := tensile.Ref("test").To("dep")
	provider := testNode{Name: "provider", Provide: []tensile.NodeRef{ref}}
	depender := testNode{Name: "depender", DependOn: []tensile.NodeRef{ref}}
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

	work, notifierID, _ := buildNotifiedWork(t)

	items := work.Chan(t.Context())

	first := <-items
	require.NoError(t, first.Err)
	require.Equal(t, notifierID, first.Node.ID())
	work.MarkDone(first.Node, false)

	item, open := <-items
	assert.False(t, open, "handler without executed notifier must be skipped: %+v", item)
}

func TestWork_ChanYieldsHandlerAfterNotifierExecuted(t *testing.T) {
	t.Parallel()

	work, notifierID, handler := buildNotifiedWork(t)

	items := work.Chan(t.Context())

	first := <-items
	require.NoError(t, first.Err)
	require.Equal(t, notifierID, first.Node.ID())
	work.MarkDone(first.Node, true)

	second := <-items
	require.NoError(t, second.Err)
	assert.Equal(t, handler.ID(), second.Node.ID())
}
