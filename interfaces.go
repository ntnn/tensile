package tensile

import "context"

type Validator interface {
	Validate(context.Context) error
}

type ValidatorCtx interface {
	Validate(Context) error
}

type Provider interface {
	Provides() ([]NodeRef, error)
}

type Depender interface {
	DependsOn() ([]NodeRef, error)
}

type Executor interface {
	NeedsExecution(context.Context) (bool, error)
	Execute(context.Context) error
}

type ExecutorCtx interface {
	NeedsExecution(Context) (bool, error)
	Execute(Context) error
}
