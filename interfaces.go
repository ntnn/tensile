package tensile

import (
	"fmt"
)

// interfaces commonly used in tensile

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
