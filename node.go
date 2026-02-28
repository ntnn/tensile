package tensile

import (
	"encoding/json"
	"fmt"
	"hash/fnv"
)

type Node struct {
	wrapped any
	id      int64
}

func NewNode(input any) (*Node, error) {
	if node, ok := input.(*Node); ok {
		return node, nil
	}

	n := new(Node)
	n.wrapped = input

	id, err := hash(input)
	if err != nil {
		return nil, fmt.Errorf("failed to hash input: %w", err)
	}
	n.id = id

	return n, nil
}

func hash(input any) (int64, error) {
	b, err := json.Marshal(input)
	if err != nil {
		return 0, fmt.Errorf("failed to marshal input: %w", err)
	}

	h := fnv.New64a()
	if _, err := h.Write(b); err != nil {
		return 0, fmt.Errorf("failed to write to hash: %w", err)
	}
	return int64(h.Sum64()), nil
}

func (n *Node) ID() int64 {
	return n.id
}

func (n *Node) Validate(ctx Context) error {
	if validator, ok := n.wrapped.(ValidatorCtx); ok {
		return validator.Validate(ctx)
	}
	if validator, ok := n.wrapped.(Validator); ok {
		return validator.Validate(ctx)
	}
	return nil
}

func (n *Node) Provides() ([]NodeRef, error) {
	provider, ok := n.wrapped.(Provider)
	if !ok {
		return nil, nil
	}
	return provider.Provides()
}

func (n *Node) DependsOn() ([]NodeRef, error) {
	depender, ok := n.wrapped.(Depender)
	if !ok {
		return nil, nil
	}
	return depender.DependsOn()
}

func (n *Node) NeedsExecution(ctx Context) (bool, error) {
	if executor, ok := n.wrapped.(ExecutorCtx); ok {
		return executor.NeedsExecution(ctx)
	}
	if executor, ok := n.wrapped.(Executor); ok {
		return executor.NeedsExecution(ctx)
	}
	return true, nil
}

func (n *Node) Execute(ctx Context) error {
	if executor, ok := n.wrapped.(ExecutorCtx); ok {
		return executor.Execute(ctx)
	}
	if executor, ok := n.wrapped.(Executor); ok {
		return executor.Execute(ctx)
	}
	return nil
}
