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
	return FormatIdentitierParts(shape, name)
}

func FormatIdentitierParts(shape Shape, name string) string {
	return fmt.Sprintf("%s[%s]", shape, name)
}

// IsCollisioner is used to detect wether two nodes with the same
// identity cause a collision.
type IsCollisioner interface {
	// IsCollision receives another node and should return an error if they
	// are colliding.
	// See the Package node for an example.
	IsCollision(other Identitier) error
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
	Nodes(facts.Facts) (chan Identitier, chan error, error)
}
