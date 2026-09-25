package queue

import "github.com/ntnn/tensile"

// NamedQueuer is the interface for values that provide a [NamedQueue].
// [Queue.Add] adds the provided queue in place of the value.
type NamedQueuer interface {
	// Queue returns the [NamedQueue] to add in place of the value.
	Queue() *NamedQueue
}

// NamedQueue is a [Queue] with an identity.
//
// It can be used to provide a set of [tensile.Node] with dependencies
// and notifications setup to add to another [Queue] or [NamedQueue].
//
// Note that a [NamedQueue] being added twice from different locations
// will error at build time. Hence they should be used to compose
// smaller parts into a larger queue rather than ensuring prerequisites.
type NamedQueue struct {
	Queue

	identity tensile.Identity
}

// NewNamed returns a new [NamedQueue] with the given identity.
func NewNamed(identity tensile.Identity) *NamedQueue {
	q := new(NamedQueue)
	q.identity = identity
	return q
}

// Identity implements [tensile.Identifier].
func (nq *NamedQueue) Identity() tensile.Identity {
	return nq.identity
}

// identities returns the identities of the queue's nodes.
func (nq *NamedQueue) identities() []tensile.Identity {
	ids := make([]tensile.Identity, len(nq.nodes))
	for i, node := range nq.nodes {
		ids[i] = node.Identity()
	}
	return ids
}
