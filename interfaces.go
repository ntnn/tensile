package tensile

type Validator interface {
	Validate() error
}

type Provider interface {
	Provides() ([]NodeRef, error)
}

type Depender interface {
	DependsOn() ([]NodeRef, error)
}

type Executor interface {
	NeedsExecution() (bool, error)
	Execute() error
}
