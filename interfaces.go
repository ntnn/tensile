package tensile

import "context"

// Validator is the interface to be satisfied by a [Node] when the
// configuration needs to be validated e.g. before execution.
type Validator interface {
	// Validate validates the configuration of the node.
	// It may be used to setup states in the node.
	Validate(ctx context.Context) error
}

// ValidatorCtx is equivalent to [Validator] but with a [Context]
// instead of a [context.Contex].
type ValidatorCtx interface {
	// Validate validates the configuration of the node.
	// It may be used to setup states in the node.
	Validate(ctx Context) error
}

// Provider is the interface to be satisfied by a [Node] when it
// provides resources, e.g. installing a package or creating a file.
type Provider interface {
	// Provides returns a list of resources the node will provide, e.g.
	// a list of packages or files.
	Provides() ([]NodeRef, error)
}

// Depender is the interface to be satisfied by a [Node] when it depends
// on other resources to be provided by other [Node]., e.g. a package
// that must be installed or a file to be ensured.
type Depender interface {
	// DependsOn returns a list of resources the node depends on, e.g.
	// packages or files.
	DependsOn() ([]NodeRef, error)
}

// Executor is the interface to be satisfied by a [Node] to be executed.
type Executor interface {
	// NeedsExecution is run before Execute. NeedsExecution must not
	// mane any modifications to the system, it must be a stateless
	// check to check if Node.Execute must be called.
	//
	// NeedsExecution is called e.g. for noop runs to check if any
	// changes are needed.
	NeedsExecution(ctx context.Context) (bool, error)
	// Execute is called for the node to make the desired change.
	Execute(ctx context.Context) error
}

// ExecutorCtx is equivalent to [Executor] but with a [Context] instead
// of a [context.Context].
type ExecutorCtx interface {
	// NeedsExecution is run before Execute. NeedsExecution must not
	// mane any modifications to the system, it must be a stateless
	// check to check if Node.Execute must be called.
	//
	// NeedsExecution is called e.g. for noop runs to check if any
	// changes are needed.
	NeedsExecution(ctx Context) (bool, error)
	// Execute is called for the node to make the desired change.
	Execute(ctx Context) error
}
