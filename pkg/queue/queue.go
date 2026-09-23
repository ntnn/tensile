package queue

import (
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
}

// New returns a new [Queue].
func New() *Queue {
	return &Queue{}
}

// Add adds values as [tensile.Node] to the queue.
func (q *Queue) Add(nodes ...tensile.Identifier) {
	q.nodes = append(q.nodes, nodes...)
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
	b := newBuild()
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
