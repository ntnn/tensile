package tensile

import (
	"fmt"

	"github.com/ntnn/tensile/facts"
)

// interfaces commonly used in tensile

// Node is a collection of interfaces a node _should_ adhere to.
type Node interface {
	Validator
	Executor
}

// Validator validates an element when adding it to a queue.
// The error will be returned by the queue if validation fails.
//
// This can be used to e.g. validate that all required options are set
// or to set internal values.
type Validator interface {
	Validate() error
}

// Identitier returns the full Identity of an element.
type Identitier interface {
	Identity() (Shape, string)
}

// FormatIdentitier formats the identity of an Identitier.
func FormatIdentitier(ident Identitier) string {
	shape, name := ident.Identity()
	return fmt.Sprintf("%s[%s]", shape, name)
}

type NeedsExecutioner interface {
	NeedsExecution(Context) (bool, error)
}

// Executor executes the element.
type Executor interface {
	// Execute executes the element.
	// The return value can be a struct to be utilized by other
	// elements.
	Execute(Context) (any, error)
}

type PreElementer interface {
	// PreElements returns the identity of elements that should be
	// executed before this element.
	PreElements() []string
}

// NodeGenerator can generate more nodes for a Queue to collect.
//
// This is primarily useful to have a single Noop node that then
// dynamically generates more nodes based on the facts of the system.
type NodeGenerator interface {
	Nodes(*facts.Facts) (chan Identitier, chan error, error)
}
