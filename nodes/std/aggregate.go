package std

import (
	"fmt"
	"strings"

	"github.com/ntnn/tensile"
)

var _ tensile.Identifier = (*Aggregate)(nil)
var _ tensile.Validator = (*Aggregate)(nil)
var _ tensile.Provider = (*Aggregate)(nil)
var _ tensile.Depender = (*Aggregate)(nil)
var _ tensile.Executor = (*Aggregate)(nil)

// Aggregate is a utility node that can chain multiple other nodes.
// For example uses see e.g. the [File] node.
type Aggregate struct {
	contained []*tensile.Node
}

// NewAggregate returns a new [Aggregate].
func NewAggregate(raw ...tensile.Identifier) *Aggregate {
	nodes := make([]*tensile.Node, len(raw))
	for i, r := range raw {
		nodes[i] = tensile.NewNode(r)
	}
	return &Aggregate{contained: nodes}
}

// Identity implements [tensile.Identifier].
// The identity is derived from the contained nodes.
func (a *Aggregate) Identity() tensile.Identity {
	contained := make([]string, len(a.contained))
	for i, node := range a.contained {
		contained[i] = node.Identity().String()
	}
	return tensile.AsIdentity("aggregate", "of", strings.Join(contained, " "))
}

// Validate implements [tensile.Validator].
func (a *Aggregate) Validate(s tensile.Wire) error {
	for i, node := range a.contained {
		if err := node.Validate(s); err != nil {
			return fmt.Errorf("error in .Validate of %d %v: %w", i, node, err)
		}
	}
	return nil
}

// Provides implements [tensile.Provides].
func (a *Aggregate) Provides() ([]tensile.Identity, error) {
	refs := []tensile.Identity{}
	for i, node := range a.contained {
		cRefs, err := node.Provides()
		if err != nil {
			return nil, fmt.Errorf("error in .Provides of %d %v: %w", i, node, err)
		}
		refs = append(refs, cRefs...)
	}
	return refs, nil
}

// DependsOn implements [tensile.DependsOn].
func (a *Aggregate) DependsOn() ([]tensile.Identity, error) {
	refs := []tensile.Identity{}
	for i, node := range a.contained {
		cRefs, err := node.DependsOn()
		if err != nil {
			return nil, fmt.Errorf("error in .DependsOn of %d %v: %w", i, node, err)
		}
		refs = append(refs, cRefs...)
	}
	return refs, nil
}

// NeedsExecution implements [tensile.Executor].
func (a *Aggregate) NeedsExecution(s tensile.Wire) (bool, error) {
	for i, node := range a.contained {
		needsExecution, err := node.NeedsExecution(s)
		if err != nil {
			return false, fmt.Errorf("error in .NeedsExecution of %d %v: %w", i, node, err)
		}
		if needsExecution {
			return true, nil
		}
	}
	return false, nil
}

// Execute implements [tensile.Executor].
func (a *Aggregate) Execute(s tensile.Wire) error {
	for i, node := range a.contained {
		needsExecution, err := node.NeedsExecution(s)
		if err != nil {
			return fmt.Errorf("error in .NeedsExecution of %d %v: %w", i, node, err)
		}
		if !needsExecution {
			continue
		}
		if err := node.Execute(s); err != nil {
			return fmt.Errorf("error in .Execute of %d %v: %w", i, node, err)
		}
	}
	return nil
}
