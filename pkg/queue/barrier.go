package queue

import "github.com/ntnn/tensile"

var _ tensile.Identifier = (*barrier)(nil)

const barrierKind = "barrier"

// barrier is a hidden node used to gate the start or end of a [tensile.Group].
type barrier struct {
	identity tensile.Identity
}

// Identity implements [tensile.Identifier].
func (b barrier) Identity() tensile.Identity {
	return b.identity
}

// barriers is the start and end [barrier] of a [tensile.Group].
type barriers struct {
	start, end barrier
}

// newBarriers returns the barriers for the named group.
func newBarriers(name string) barriers {
	return barriers{
		start: barrier{identity: tensile.AsIdentity(barrierKind, "group", name, "side", "start")},
		end:   barrier{identity: tensile.AsIdentity(barrierKind, "group", name, "side", "end")},
	}
}
