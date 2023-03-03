package gorrect

import (
	"context"
	"fmt"
)

// interfaces commonly used by engines

type Validator interface {
	Validate() error
}

type Identitier interface {
	Identity() (Shape, string)
}

func FormatIdentitier(ident Identitier) string {
	shape, name := ident.Identity()
	return fmt.Sprintf("%s[%s]", shape, name)
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
