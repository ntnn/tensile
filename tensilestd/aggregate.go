package tensilestd

import (
	"fmt"

	"github.com/ntnn/tensile"
)

var _ tensile.ValidatorCtx = (*Aggregate)(nil)
var _ tensile.Provider = (*Aggregate)(nil)
var _ tensile.Depender = (*Aggregate)(nil)
var _ tensile.ExecutorCtx = (*Aggregate)(nil)

// AggregateRef is the reference type for aggregates.
const AggregateRef = tensile.Ref("Aggregate")

// Aggregate is a utility node that can chain multiple other nodes.
// For example uses see e.g. the [File] node.
type Aggregate struct {
	contained []*tensile.Node
}

// NewAggregate eturns a new [Aggregate]. The values may be
// [tensile.Node], if they are not they will be converted.
func NewAggregate(raw ...any) (*Aggregate, error) {
	nodes := make([]*tensile.Node, len(raw))
	for i, r := range raw {
		node, err := tensile.NewNode(r)
		if err != nil {
			return nil, fmt.Errorf("error transforming input %d %q to a tensile node: %w", i, r, err)
		}
		nodes[i] = node
	}
	return &Aggregate{contained: nodes}, nil
}

// Validate implements [tensile.Validator].
func (a *Aggregate) Validate(ctx tensile.Context) error {
	for i, node := range a.contained {
		if err := node.Validate(ctx); err != nil {
			return fmt.Errorf("error in .Validate of %d %q: %w", i, node, err)
		}
	}
	return nil
}

// Provides implements [tensile.Provides].
func (a *Aggregate) Provides() ([]tensile.NodeRef, error) {
	refs := []tensile.NodeRef{}
	for i, node := range a.contained {
		cRefs, err := node.Provides()
		if err != nil {
			return nil, fmt.Errorf("error in .Provides of %d %q: %w", i, node, err)
		}
		refs = append(refs, cRefs...)
	}
	return refs, nil
}

// DependsOn implements [tensile.DependsOn].
func (a *Aggregate) DependsOn() ([]tensile.NodeRef, error) {
	refs := []tensile.NodeRef{}
	for i, node := range a.contained {
		cRefs, err := node.DependsOn()
		if err != nil {
			return nil, fmt.Errorf("error in .DependsOn of %d %q: %w", i, node, err)
		}
		refs = append(refs, cRefs...)
	}
	return refs, nil
}

// NeedsExecution implements [tensile.ExecutorCtx].
func (a *Aggregate) NeedsExecution(ctx tensile.Context) (bool, error) {
	for i, node := range a.contained {
		needsExecution, err := node.NeedsExecution(ctx)
		if err != nil {
			return false, fmt.Errorf("error in .NeedsExecution of %d %q: %w", i, node, err)
		}
		if needsExecution {
			return true, nil
		}
	}
	return false, nil
}

// Execute implements [tensile.ExecutorCtx].
func (a *Aggregate) Execute(ctx tensile.Context) error {
	for i, node := range a.contained {
		needsExecution, err := node.NeedsExecution(ctx)
		if err != nil {
			return fmt.Errorf("error in .NeedsExecution of %d %q: %w", i, node, err)
		}
		if !needsExecution {
			continue
		}
		if err := node.Execute(ctx); err != nil {
			return fmt.Errorf("error in .Execute of %d %q: %w", i, node, err)
		}
	}
	return nil
}
