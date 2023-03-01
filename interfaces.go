package gorrect

import (
	"context"
)

// interfaces commonly used by engines

type Validator interface {
	Validate() error
}

type Identitier interface {
	Identity() (Identity, string)
}

type NoIdentityClasher interface {
	NoIdentityClash() bool
}

type Executor interface {
	Execute(context.Context) error
}

type PreElementer interface {
	// PreElements returns the identity of elements that should be
	// executed before this element.
	PreElements() []string
}
