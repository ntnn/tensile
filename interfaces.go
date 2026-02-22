package tensile

type Validator interface {
	Validate() error
}

type Provider interface {
	Provides() ([]string, error)
}

type Depender interface {
	DependsOn() ([]string, error)
}

type Executor interface {
	NeedsExecution() (bool, error)
	Execute() error
}
