package tensile

import (
	"fmt"
)

// interfaces commonly used in tensile

// Node is a collection of interfaces a node _should_ adhere to.
// type Node interface {
// 	Validator
// 	Executor
// }

type PreElementer interface {
	// PreElements returns the identity of elements that should be
	// executed before this element.
	PreElements() []string
}

// NodeGenerator can return more nodes for a Queue to collect.
//
// This is primarily useful to have a single Noop node that dynamically
// generates more nodes based on configuration.
type NodeGenerator interface {
	Nodes() ([]Identitier, error)
}
