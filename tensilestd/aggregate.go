package tensilestd

import (
	"fmt"

	"github.com/ntnn/tensile"
)

var _ tensile.Validator = (*Aggregate)(nil)
var _ tensile.Provider = (*Aggregate)(nil)
var _ tensile.Depender = (*Aggregate)(nil)
var _ tensile.Executor = (*Aggregate)(nil)

const AggregateRef = tensile.Ref("Aggregate")

type Aggregate struct {
	Contained []any
}

func (a *Aggregate) Validate() error {
	for i, c := range a.Contained {
		validator, ok := c.(tensile.Validator)
		if !ok {
			continue
		}
		if err := validator.Validate(); err != nil {
			return fmt.Errorf("error in .Validate of %d %q: %w", i, c, err)
		}
	}
	return nil
}

func (a *Aggregate) Provides() ([]tensile.NodeRef, error) {
	refs := []tensile.NodeRef{}
	for i, c := range a.Contained {
		provider, ok := c.(tensile.Provider)
		if !ok {
			continue
		}
		cRefs, err := provider.Provides()
		if err != nil {
			return nil, fmt.Errorf("error in .Provides of %d %q: %w", i, c, err)
		}
		refs = append(refs, cRefs...)
	}
	return refs, nil
}

func (a *Aggregate) DependsOn() ([]tensile.NodeRef, error) {
	refs := []tensile.NodeRef{}
	for i, c := range a.Contained {
		depender, ok := c.(tensile.Depender)
		if !ok {
			continue
		}
		cRefs, err := depender.DependsOn()
		if err != nil {
			return nil, fmt.Errorf("error in .DependsOn of %d %q: %w", i, c, err)
		}
		refs = append(refs, cRefs...)
	}
	return refs, nil
}

func (a *Aggregate) NeedsExecution() (bool, error) {
	for i, c := range a.Contained {
		executor, ok := c.(tensile.Executor)
		if !ok {
			continue
		}
		needsExecution, err := executor.NeedsExecution()
		if err != nil {
			return false, fmt.Errorf("error in .NeedsExecution of %d %q: %w", i, c, err)
		}
		if needsExecution {
			return true, nil
		}
	}
	return false, nil
}

func (a *Aggregate) Execute() error {
	for i, c := range a.Contained {
		executor, ok := c.(tensile.Executor)
		if !ok {
			continue
		}
		needsExecution, err := executor.NeedsExecution()
		if err != nil {
			return fmt.Errorf("error in .NeedsExecution of %d %q: %w", i, c, err)
		}
		if !needsExecution {
			continue
		}
		if err := executor.Execute(); err != nil {
			return fmt.Errorf("error in .Execute of %d %q: %w", i, c, err)
		}
	}
	return nil
}
