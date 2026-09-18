package tensile

// Node is a single step to be executed by an engine.
type Node struct {
	wrapped  any
	identity Identity
}

// NewNode wraps an [Identifier] into a [Node].
func NewNode(input Identifier) *Node {
	if node, ok := input.(*Node); ok {
		return node
	}
	if handler, ok := input.(*Handler); ok {
		return &handler.Node
	}

	return &Node{
		wrapped:  input,
		identity: input.Identity(),
	}
}

// Identity returns the identity of the wrapped node.
func (n *Node) Identity() Identity {
	return n.identity
}

// Validate calls .Validate on the wrapped node if it implements it.
func (n *Node) Validate(wire Wire) error {
	if validator, ok := n.wrapped.(Validator); ok {
		return validator.Validate(wire)
	}
	return nil
}

// Conflicts calls .Conflicts on the wrapped node if it implements it.
func (n *Node) Conflicts() ([]Identity, error) {
	conflictor, ok := n.wrapped.(Conflictor)
	if !ok {
		return nil, nil
	}
	return conflictor.Conflicts()
}

// DependsOn calls .DependsOn on the wrapped node if it implements it.
func (n *Node) DependsOn() ([]Identity, error) {
	depender, ok := n.wrapped.(Depender)
	if !ok {
		return nil, nil
	}
	return depender.DependsOn()
}

// Notifies calls .Notifies on the wrapped node if it implements it.
func (n *Node) Notifies() ([]Identity, error) {
	notifier, ok := n.wrapped.(Notifier)
	if !ok {
		return nil, nil
	}
	return notifier.Notifies()
}

// Report calls .Report on the wrapped node if it implements it.
// The bool reports whether the wrapped node is a [Reporter].
func (n *Node) Report(wire Wire) (any, bool, error) {
	reporter, ok := n.wrapped.(Reporter)
	if !ok {
		return nil, false, nil
	}
	output, err := reporter.Report(wire)
	return output, true, err
}

// NeedsExecution calls .NeedsExecution on the wrapped node if it implements it.
func (n *Node) NeedsExecution(wire Wire) (bool, error) {
	if executor, ok := n.wrapped.(Executor); ok {
		return executor.NeedsExecution(wire)
	}
	return true, nil
}

// Execute calls .Execute on the wrapped node if it implements it.
func (n *Node) Execute(wire Wire) error {
	if executor, ok := n.wrapped.(Executor); ok {
		return executor.Execute(wire)
	}
	return nil
}
