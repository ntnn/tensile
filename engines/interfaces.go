package engines

import "context"

// interfaces commonly used by engines

type Validator interface {
	// Validates an element.
	Validate() error
}

type Identitier interface {
	Identity() string
}

type Executor interface {
	Execute(context.Context) error
}

type PreElementer interface {
	// PreElements returns the identity of elements that should be
	// executed before this element.
	PreElements() []string
}
