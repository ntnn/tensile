package queue

import (
	"fmt"
	"slices"

	"github.com/ntnn/tensile"
	"github.com/ntnn/tensile/pkg/graph"
)

// notification records which nodes notify a handler.
type notification struct {
	handler   tensile.Identity
	notifiers []tensile.Identity
}

// Queue records nodes, dependencies and notifications.
type Queue struct {
	nodes    []tensile.Identifier
	deps     []graph.Edge[tensile.Identity]
	notifies []notification
	// subqueues maps [NamedQueue] identities to their node identities
	subqueues map[tensile.Identity][]tensile.Identity
	// duplicates tracks duplicate [NamedQueue] identities since
	// subqueues is a map and .Add cannot error
	duplicates []tensile.Identity
	// breadcrumbs maps node identities to the [NamedQueue] chain they
	// were added through, innermost first
	breadcrumbs map[tensile.Identity][]tensile.Identity
}

// New returns a new [Queue].
func New() *Queue {
	return &Queue{}
}

// Add adds values as [tensile.Node] to the queue.
func (q *Queue) Add(nodes ...tensile.Identifier) {
	for _, node := range nodes {
		if nq, ok := node.(*NamedQueue); ok {
			q.addNamed(nq)
			continue
		}
		q.nodes = append(q.nodes, node)
	}
}

// addNamed dissolves the named queue, merging its intents and
// memberships and extending each member's breadcrumb with the queue.
func (q *Queue) addNamed(nq *NamedQueue) {
	if q.subqueues == nil {
		q.subqueues = map[tensile.Identity][]tensile.Identity{}
		q.breadcrumbs = map[tensile.Identity][]tensile.Identity{}
	}

	q.nodes = append(q.nodes, nq.nodes...)
	q.deps = append(q.deps, nq.deps...)
	q.notifies = append(q.notifies, nq.notifies...)
	q.duplicates = append(q.duplicates, nq.duplicates...)

	// map existing subqueues from subqueue
	for identity, subqueues := range nq.subqueues {
		if _, exists := q.subqueues[identity]; exists {
			q.duplicates = append(q.duplicates, identity)
			continue
		}
		q.subqueues[identity] = subqueues
	}
	// add the subqueue itself
	identities := nq.identities()
	if _, exists := q.subqueues[nq.Identity()]; exists {
		q.duplicates = append(q.duplicates, nq.Identity())
	}
	q.subqueues[nq.Identity()] = identities

	// and add the breadcrumbs to see where nodes came from
	for _, identity := range identities {
		if _, exists := q.breadcrumbs[identity]; exists {
			// keep the first breadcrumb, duplicate nodes error at build
			continue
		}
		q.breadcrumbs[identity] = append(slices.Clone(nq.breadcrumbs[identity]), nq.Identity())
	}
}

// DependsOn makes node dependent on each node in dependsOn.
func (q *Queue) DependsOn(node tensile.Identifier, dependsOn ...tensile.Identifier) {
	for _, dep := range dependsOn {
		q.deps = append(
			q.deps,
			graph.Edge[tensile.Identity]{
				From: dep.Identity(),
				To:   node.Identity(),
			},
		)
	}
}

// RequiredBy is like [Queue.DependsOn] but makes each requiredBy dependent on node.
func (q *Queue) RequiredBy(node tensile.Identifier, requiredBy ...tensile.Identifier) {
	for _, req := range requiredBy {
		q.deps = append(q.deps, graph.Edge[tensile.Identity]{
			From: node.Identity(),
			To:   req.Identity(),
		})
	}
}

// NotifiedBy adds the notifiers as notifiers for handler.
func (q *Queue) NotifiedBy(handler *tensile.Handler, notifiers ...tensile.Identifier) {
	identities := make([]tensile.Identity, len(notifiers))
	for i, notifier := range notifiers {
		identities[i] = notifier.Identity()
	}
	q.notifies = append(q.notifies, notification{
		handler:   handler.Identity(),
		notifiers: identities,
	})
}

// Build returns a [Work] with the added [tensile.Node], dependencies and notifications.
func (q *Queue) Build() (*Work, error) {
	if len(q.duplicates) > 0 {
		return nil, fmt.Errorf("multiple NamedQueue using the same identity in graph: %v", q.duplicates)
	}

	b := newBuild()
	if err := b.addSubqueues(q.subqueues, q.breadcrumbs); err != nil {
		return nil, err
	}
	if err := b.addNodes(q.nodes); err != nil {
		return nil, err
	}
	if err := b.addDependencies(q.deps); err != nil {
		return nil, err
	}
	if err := b.addNotifies(q.notifies); err != nil {
		return nil, err
	}
	if err := b.implicit(); err != nil {
		return nil, err
	}
	return b.work()
}
