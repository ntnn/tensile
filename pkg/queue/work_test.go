package queue

import (
	"context"
	"testing"
	"time"

	"github.com/ntnn/tensile"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type testNode struct {
	Name     string
	Conflict []tensile.Identity
	DependOn []tensile.Identity
}

func (n testNode) Identity() tensile.Identity {
	return tensile.AsIdentity("test", "name", n.Name)
}

func (n testNode) Conflicts() ([]tensile.Identity, error) {
	return n.Conflict, nil
}

func (n testNode) DependsOn() ([]tensile.Identity, error) {
	return n.DependOn, nil
}

func buildWork(t *testing.T, nodes ...tensile.Identifier) *Work {
	t.Helper()
	q := New()
	q.Add(nodes...)
	work, err := q.Build()
	require.NoError(t, err)
	return work
}

// buildNotifiedWork builds a Work with a single notifier node and a
// handler notified by it, returning the work and the notifier's identity.
func buildNotifiedWork(t *testing.T) (*Work, tensile.Identity, *tensile.Handler) {
	t.Helper()

	node := testNode{Name: "notifier"}
	handler := tensile.NewHandler(testNode{Name: "handler"})

	q := New()
	q.Add(node, handler)
	q.NotifiedBy(handler, node)

	work, err := q.Build()
	require.NoError(t, err)

	return work, nodeIdentity(t, node), handler
}

func nodeIdentity(t *testing.T, input tensile.Identifier) tensile.Identity {
	t.Helper()
	return tensile.NewNode(input).Identity()
}

func TestQueue_BuildErrorsOnSharedConflictIdentity(t *testing.T) {
	t.Parallel()

	ref := tensile.AsIdentity("testres", "name", "shared")
	a := testNode{Name: "a", Conflict: []tensile.Identity{ref}}
	b := testNode{Name: "b", Conflict: []tensile.Identity{ref}}

	q := New()
	q.Add(a, b)
	_, err := q.Build()
	require.ErrorContains(t, err, "already claimed")
}

func TestQueue_BuildErrorsOnConflictWithNodeIdentity(t *testing.T) {
	t.Parallel()

	a := testNode{Name: "a"}
	b := testNode{Name: "b", Conflict: []tensile.Identity{nodeIdentity(t, a)}}

	q := New()
	q.Add(a, b)
	_, err := q.Build()
	require.ErrorContains(t, err, "already claimed")
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
	provider := testNode{Name: "provider", Conflict: []tensile.Identity{ref}}
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
	provider := testNode{Name: "provider", Conflict: []tensile.Identity{ref}}
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

	q := New()
	q.Add(first, second)
	q.DependsOn(second, first)
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

// serialNode is a testNode with serialization keys.
type serialNode struct {
	testNode

	Keys []string
}

func (n serialNode) SerializesOn() []string {
	return n.Keys
}

func TestWork_ChanSerializesSameKey(t *testing.T) {
	t.Parallel()

	a := serialNode{Name: "a", Keys: []string{"mgr"}}
	b := serialNode{Name: "b", Keys: []string{"mgr"}}
	work := buildWork(t, a, b)

	items := work.Chan(t.Context())

	first := <-items
	require.NoError(t, first.Err)
	require.NotNil(t, first.Node)

	select {
	case item := <-items:
		t.Fatalf("node yielded while %s holds the key: %+v", first.Node.Identity(), item)
	case <-time.After(50 * time.Millisecond):
	}

	work.MarkDone(first.Node, true)

	second := <-items
	require.NoError(t, second.Err)
	require.NotNil(t, second.Node)
	assert.NotEqual(t, first.Node.Identity(), second.Node.Identity())

	work.MarkDone(second.Node, true)

	_, open := <-items
	assert.False(t, open, "channel must be closed after all nodes were yielded")
}

func TestWork_ChanSerializesMultiKey(t *testing.T) {
	t.Parallel()

	// commit-all pattern: the multi-key node contends with holders of
	// either key
	for name, other := range map[string]serialNode{
		"coarse": {Name: "coarse", Keys: []string{"uci"}},
		"fine":   {Name: "fine", Keys: []string{"uci-a"}},
	} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			all := serialNode{Name: "all", Keys: []string{"uci", "uci-a"}}
			work := buildWork(t, all, other)

			items := work.Chan(t.Context())

			first := <-items
			require.NoError(t, first.Err)
			require.NotNil(t, first.Node)

			select {
			case item := <-items:
				t.Fatalf("node yielded while %s holds a shared key: %+v", first.Node.Identity(), item)
			case <-time.After(50 * time.Millisecond):
			}

			work.MarkDone(first.Node, true)

			second := <-items
			require.NoError(t, second.Err)
			require.NotNil(t, second.Node)
			work.MarkDone(second.Node, true)

			_, open := <-items
			assert.False(t, open, "channel must be closed after all nodes were yielded")
		})
	}
}

func TestWork_ChanOverlapsDifferentKeys(t *testing.T) {
	t.Parallel()

	a := serialNode{Name: "a", Keys: []string{"pacman"}}
	b := serialNode{Name: "b", Keys: []string{"opkg"}}
	work := buildWork(t, a, b)

	items := work.Chan(t.Context())

	// both nodes must be yielded without any MarkDone in between
	first := <-items
	require.NoError(t, first.Err)
	require.NotNil(t, first.Node)

	second := <-items
	require.NoError(t, second.Err)
	require.NotNil(t, second.Node)
	assert.NotEqual(t, first.Node.Identity(), second.Node.Identity())
}

func TestWork_ChanOverlapsEmptyKeys(t *testing.T) {
	t.Parallel()

	a := serialNode{Name: "a"}
	b := testNode{Name: "b"}
	work := buildWork(t, a, b)

	items := work.Chan(t.Context())

	first := <-items
	require.NoError(t, first.Err)
	require.NotNil(t, first.Node)

	second := <-items
	require.NoError(t, second.Err)
	require.NotNil(t, second.Node)
}

// namedQueue builds a NamedQueue with the given name and nodes.
func namedQueue(t *testing.T, name string, nodes ...tensile.Identifier) *NamedQueue {
	t.Helper()
	nq := NewNamed(tensile.AsIdentity("queue", "name", name))
	nq.Add(nodes...)
	return nq
}

// notifyNode is a node notifying targets.
type notifyNode struct {
	Name   string
	Notify []tensile.Identity
}

func (n notifyNode) Identity() tensile.Identity {
	return tensile.AsIdentity("test", "name", n.Name)
}

func (n notifyNode) Notifies() ([]tensile.Identity, error) {
	return n.Notify, nil
}

func TestQueue_AddDissolvesNamedQueue(t *testing.T) {
	t.Parallel()

	a := testNode{Name: "a"}
	b := testNode{Name: "b"}
	work := buildWork(t, namedQueue(t, "sub", a, b))

	seen := map[tensile.Identity]bool{}
	for item := range work.Chan(t.Context()) {
		require.NoError(t, item.Err)
		seen[item.Node.Identity()] = true
		work.MarkDone(item.Node, true)
	}
	want := map[tensile.Identity]bool{
		nodeIdentity(t, a): true,
		nodeIdentity(t, b): true,
	}
	assert.Equal(t, want, seen, "member nodes must be yielded, the subqueue must not")
}

func TestQueue_DependsOnNamedQueueWaitsForAllMembers(t *testing.T) {
	t.Parallel()

	a := testNode{Name: "a"}
	b := testNode{Name: "b"}
	sub := namedQueue(t, "sub", a, b)
	depender := testNode{Name: "depender"}

	q := New()
	q.Add(sub, depender)
	q.DependsOn(depender, sub)
	work, err := q.Build()
	require.NoError(t, err)

	items := work.Chan(t.Context())

	for range 2 {
		item := <-items
		require.NoError(t, item.Err)
		require.NotEqual(t, nodeIdentity(t, depender), item.Node.Identity(),
			"depender must wait for all subqueue members")
		work.MarkDone(item.Node, true)
	}

	item := <-items
	require.NoError(t, item.Err)
	assert.Equal(t, nodeIdentity(t, depender), item.Node.Identity())
}

func TestQueue_NamedQueueDependsOnNodeGatesAllMembers(t *testing.T) {
	t.Parallel()

	a := testNode{Name: "a"}
	sub := namedQueue(t, "sub", a)
	dep := testNode{Name: "dep"}

	q := New()
	q.Add(sub, dep)
	q.DependsOn(sub, dep)
	work, err := q.Build()
	require.NoError(t, err)

	items := work.Chan(t.Context())

	item := <-items
	require.NoError(t, item.Err)
	require.Equal(t, nodeIdentity(t, dep), item.Node.Identity(),
		"members must wait for the subqueue's dependency")
	work.MarkDone(item.Node, true)

	item = <-items
	require.NoError(t, item.Err)
	assert.Equal(t, nodeIdentity(t, a), item.Node.Identity())
}

func TestQueue_NestedNamedQueueDependencyWaitsForAllMembers(t *testing.T) {
	t.Parallel()

	a := testNode{Name: "a"}
	inner := namedQueue(t, "inner", a)
	b := testNode{Name: "b"}
	outer := namedQueue(t, "outer", inner, b)
	depender := testNode{Name: "depender"}

	q := New()
	q.Add(outer, depender)
	q.DependsOn(depender, outer)
	work, err := q.Build()
	require.NoError(t, err)

	items := work.Chan(t.Context())

	for range 2 {
		item := <-items
		require.NoError(t, item.Err)
		require.NotEqual(t, nodeIdentity(t, depender), item.Node.Identity(),
			"depender must wait for members of nested subqueues")
		work.MarkDone(item.Node, true)
	}

	item := <-items
	require.NoError(t, item.Err)
	assert.Equal(t, nodeIdentity(t, depender), item.Node.Identity())
}

func TestQueue_NotifiedByNamedQueueFiresOnMemberExecution(t *testing.T) {
	t.Parallel()

	a := testNode{Name: "a"}
	sub := namedQueue(t, "sub", a)
	handler := tensile.NewHandler(testNode{Name: "handler"})

	q := New()
	q.Add(sub, handler)
	q.NotifiedBy(handler, sub)
	work, err := q.Build()
	require.NoError(t, err)

	items := work.Chan(t.Context())

	item := <-items
	require.NoError(t, item.Err)
	require.Equal(t, nodeIdentity(t, a), item.Node.Identity())
	work.MarkDone(item.Node, true)

	item = <-items
	require.NoError(t, item.Err)
	require.NotNil(t, item.Node, "handler must be yielded after a member executed")
	assert.Equal(t, handler.Identity(), item.Node.Identity())
	work.MarkDone(item.Node, true)
}

func TestQueue_NotifiedByNamedQueueSkippedWithoutExecution(t *testing.T) {
	t.Parallel()

	a := testNode{Name: "a"}
	sub := namedQueue(t, "sub", a)
	handler := tensile.NewHandler(testNode{Name: "handler"})

	q := New()
	q.Add(sub, handler)
	q.NotifiedBy(handler, sub)
	work, err := q.Build()
	require.NoError(t, err)

	items := work.Chan(t.Context())

	item := <-items
	require.NoError(t, item.Err)
	require.Equal(t, nodeIdentity(t, a), item.Node.Identity())
	work.MarkDone(item.Node, false)

	got, open := <-items
	assert.False(t, open, "handler without executed member must be skipped: %+v", got)
}

func TestQueue_DeclaredDependencyOnNamedQueue(t *testing.T) {
	t.Parallel()

	a := testNode{Name: "a"}
	sub := namedQueue(t, "sub", a)
	depender := testNode{
		Name:     "depender",
		DependOn: []tensile.Identity{sub.Identity()},
	}
	work := buildWork(t, sub, depender)

	items := work.Chan(t.Context())

	item := <-items
	require.NoError(t, item.Err)
	require.Equal(t, nodeIdentity(t, a), item.Node.Identity(),
		"member must be yielded before the declarative depender")
	work.MarkDone(item.Node, true)

	item = <-items
	require.NoError(t, item.Err)
	assert.Equal(t, nodeIdentity(t, depender), item.Node.Identity())
}

func TestQueue_DependsOnClaimResolvesToClaimer(t *testing.T) {
	t.Parallel()

	ref := tensile.AsIdentity("testres", "name", "claimdep")
	provider := testNode{Name: "provider", Conflict: []tensile.Identity{ref}}
	depender := testNode{Name: "depender"}

	q := New()
	q.Add(provider, depender)
	q.DependsOn(depender, ref)
	work, err := q.Build()
	require.NoError(t, err)

	items := work.Chan(t.Context())

	item := <-items
	require.NoError(t, item.Err)
	require.Equal(t, nodeIdentity(t, provider), item.Node.Identity(),
		"manual dependency on a claim must resolve to the claiming node")
	work.MarkDone(item.Node, true)

	item = <-items
	require.NoError(t, item.Err)
	assert.Equal(t, nodeIdentity(t, depender), item.Node.Identity())
}

func TestQueue_ImplicitNotifyClaimedTargetOrdersClaimer(t *testing.T) {
	t.Parallel()

	target := tensile.AsIdentity("testres", "name", "notifytarget")
	claimer := testNode{Name: "claimer", Conflict: []tensile.Identity{target}}
	notifier := notifyNode{
		Name:   "notifier",
		Notify: []tensile.Identity{target},
	}

	q := New()
	q.Add(claimer, notifier)
	work, err := q.Build()
	require.NoError(t, err)

	items := work.Chan(t.Context())

	item := <-items
	require.NoError(t, item.Err)
	require.Equal(t, nodeIdentity(t, notifier), item.Node.Identity(),
		"claimer of a notified identity must wait for the notifier")
	work.MarkDone(item.Node, true)

	item = <-items
	require.NoError(t, item.Err)
	assert.Equal(t, nodeIdentity(t, claimer), item.Node.Identity())
}

func TestQueue_BuildErrorsOnUnknownDependency(t *testing.T) {
	t.Parallel()

	a := testNode{Name: "a"}
	missing := testNode{Name: "missing"}

	q := New()
	q.Add(a)
	q.DependsOn(a, missing)
	_, err := q.Build()
	assert.ErrorContains(t, err, "has not been added",
		"a manual dependency on an unknown node must error, not vanish")
}

func TestQueue_BuildErrorsOnUnknownNotifier(t *testing.T) {
	t.Parallel()

	handler := tensile.NewHandler(testNode{Name: "handler"})
	missing := testNode{Name: "missing"}

	q := New()
	q.Add(handler)
	q.NotifiedBy(handler, missing)
	_, err := q.Build()
	assert.ErrorContains(t, err, "has not been added")
}

func TestQueue_BuildErrorsOnDuplicateMembership(t *testing.T) {
	t.Parallel()

	a := testNode{Name: "a"}
	first := namedQueue(t, "first", a)
	second := namedQueue(t, "second", a)

	q := New()
	q.Add(first, second)
	_, err := q.Build()
	require.Error(t, err, "the same node in two subqueues must error")
	require.ErrorContains(t, err, "already claimed")
	assert.ErrorContains(t, err, `queue[name="first"]`,
		"error must name the first membership")
}

func TestQueue_BuildErrorsOnDuplicateMembershipNestedBreadcrumb(t *testing.T) {
	t.Parallel()

	a := testNode{Name: "a"}
	inner := namedQueue(t, "inner", a)
	outer := namedQueue(t, "outer", inner)
	other := namedQueue(t, "other", a)

	q := New()
	q.Add(outer, other)
	_, err := q.Build()
	require.Error(t, err)
	require.ErrorContains(t, err, `queue[name="inner"]`,
		"breadcrumb must name the innermost subqueue")
	assert.ErrorContains(t, err, `queue[name="outer"]`,
		"breadcrumb must name the parent subqueue")
}

func TestQueue_BuildErrorsOnNodeCollidingWithNamedQueue(t *testing.T) {
	t.Parallel()

	a := testNode{Name: "a"}
	sub := namedQueue(t, "sub", a)
	collider := tensile.NewNode(sub.Identity())

	q := New()
	q.Add(sub, collider)
	_, err := q.Build()
	assert.ErrorContains(t, err, "already claimed")
}

func TestQueue_BuildErrorsOnNamedQueueAddedTwice(t *testing.T) {
	t.Parallel()

	a := testNode{Name: "a"}
	sub := namedQueue(t, "sub", a)

	q := New()
	q.Add(sub, sub)
	_, err := q.Build()
	assert.Error(t, err, "the same subqueue added twice must error")
}

func TestQueue_BuildErrorsOnNotifyingNamedQueue(t *testing.T) {
	t.Parallel()

	a := testNode{Name: "a"}
	sub := namedQueue(t, "sub", a)
	notifier := notifyNode{
		Name:   "notifier",
		Notify: []tensile.Identity{sub.Identity()},
	}

	q := New()
	q.Add(sub, notifier)
	_, err := q.Build()
	assert.Error(t, err, "subqueues cannot be notified")
}

func TestQueue_BreadcrumbsSimpleNesting(t *testing.T) {
	t.Parallel()

	a := testNode{Name: "a"}
	b := testNode{Name: "b"}
	inner := namedQueue(t, "inner", a)
	outer := namedQueue(t, "outer", inner, b)

	q := New()
	q.Add(outer)

	assert.Equal(t,
		map[tensile.Identity][]tensile.Identity{
			inner.Identity(): {a.Identity()},
			outer.Identity(): {a.Identity(), b.Identity()},
		},
		q.subqueues,
	)
	assert.Equal(t,
		map[tensile.Identity][]tensile.Identity{
			a.Identity(): {inner.Identity(), outer.Identity()},
			b.Identity(): {outer.Identity()},
		},
		q.breadcrumbs,
		"breadcrumbs must chain innermost first",
	)

	_, err := q.Build()
	require.NoError(t, err)
}

func TestQueue_BreadcrumbsDeepNesting(t *testing.T) {
	t.Parallel()

	a := testNode{Name: "a"}
	inner := namedQueue(t, "inner", a)
	mid := namedQueue(t, "mid", inner)
	outer := namedQueue(t, "outer", mid)

	q := New()
	q.Add(outer)

	assert.Equal(t,
		map[tensile.Identity][]tensile.Identity{
			inner.Identity(): {a.Identity()},
			mid.Identity():   {a.Identity()},
			outer.Identity(): {a.Identity()},
		},
		q.subqueues,
	)
	assert.Equal(t,
		map[tensile.Identity][]tensile.Identity{
			a.Identity(): {inner.Identity(), mid.Identity(), outer.Identity()},
		},
		q.breadcrumbs,
		"breadcrumbs must chain innermost first",
	)

	_, err := q.Build()
	require.NoError(t, err)
}

func TestQueue_BreadcrumbsDisjointDuplicate(t *testing.T) {
	t.Parallel()

	a := testNode{Name: "a"}
	first := namedQueue(t, "first", a)
	second := namedQueue(t, "second", a)

	q := New()
	q.Add(first, second)

	assert.Equal(t,
		map[tensile.Identity][]tensile.Identity{
			a.Identity(): {first.Identity()},
		},
		q.breadcrumbs,
		"the first membership must win",
	)

	_, err := q.Build()
	require.ErrorContains(t, err, "already claimed")
	assert.ErrorContains(t, err, `queue[name="first"]`, "error must name the first membership")
}

func TestQueue_BreadcrumbsSiblingsSharingNode(t *testing.T) {
	t.Parallel()

	a := testNode{Name: "a"}
	inner1 := namedQueue(t, "inner1", a)
	inner2 := namedQueue(t, "inner2", a)
	outer := namedQueue(t, "outer", inner1, inner2)

	q := New()
	q.Add(outer)

	assert.Equal(t,
		map[tensile.Identity][]tensile.Identity{
			a.Identity(): {inner1.Identity(), outer.Identity()},
		},
		q.breadcrumbs,
		"the first membership must win",
	)

	_, err := q.Build()
	require.ErrorContains(t, err, "already claimed")
	assert.ErrorContains(t, err, `queue[name="inner1"]->queue[name="outer"]`)
}

func TestQueue_BreadcrumbsNodeDirectAndNested(t *testing.T) {
	t.Parallel()

	a := testNode{Name: "a"}
	inner := namedQueue(t, "inner", a)
	outer := namedQueue(t, "outer", inner, a)

	q := New()
	q.Add(outer)

	_, err := q.Build()
	require.ErrorContains(t, err, "already claimed")
	assert.ErrorContains(t, err, `queue[name="inner"]->queue[name="outer"]`,
		"the directly added duplicate is attributed to the subqueue membership",
	)
}

func TestQueue_BreadcrumbsQueueAddedDirectlyAndNested(t *testing.T) {
	t.Parallel()

	a := testNode{Name: "a"}
	inner := namedQueue(t, "inner", a)
	outer := namedQueue(t, "outer", inner)

	q := New()
	q.Add(outer, inner)

	_, err := q.Build()
	require.ErrorContains(t, err, "already claimed")
	assert.ErrorContains(t, err, `queue[name="inner"]->queue[name="outer"]`)
}
