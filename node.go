package tensile

// Node is a single step to be executed by an engine.
type Node struct {
	wrapped  any
	identity Identity

	// when gates the node, the zero value is always enabled
	when Condition
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

// Condition gates a [Node].
type Condition struct {
	// Reads are dependencies the condition may read.
	Reads []Identity
	// Cond decides whether the node runs.
	// nil means always enabled.
	Cond func(Wire) (bool, error)
}

// Cond returns a [Condition] from fn and the dependencies it reads.
func Cond(fn func(Wire) (bool, error), reads ...Identity) Condition {
	return Condition{
		Reads: reads,
		Cond:  fn,
	}
}

// When wraps input into a [Node] whose execution is gated by cond.
// If the condition returns false the node's validation and execution is skipped.
// [Condition.Reads] are added to the node's dependencies.
func When(cond Condition, input Identifier) *Node {
	node := NewNode(input)
	node.when = cond
	return node
}

// enabled evaluates the when condition, nil means enabled.
func (n *Node) enabled(wire Wire) (bool, error) {
	if n.when.Cond == nil {
		return true, nil
	}
	return n.when.Cond(wire)
}

// Identity returns the identity of the wrapped node.
func (n *Node) Identity() Identity {
	return n.identity
}

// Validate calls .Validate on the wrapped node if it implements it.
// A disabled node validates nothing.
func (n *Node) Validate(wire Wire) error {
	enabled, err := n.enabled(wire)
	if err != nil {
		return err
	}
	if !enabled {
		return nil
	}
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
		return n.when.Reads, nil
	}
	deps, err := depender.DependsOn()
	if err != nil {
		return nil, err
	}
	return append(deps, n.when.Reads...), nil
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
	enabled, err := n.enabled(wire)
	if err != nil {
		return false, err
	}
	if !enabled {
		return false, nil
	}
	if executor, ok := n.wrapped.(Executor); ok {
		return executor.NeedsExecution(wire)
	}
	return true, nil
}

// Execute calls .Execute on the wrapped node if it implements it.
func (n *Node) Execute(wire Wire) error {
	enabled, err := n.enabled(wire)
	if err != nil {
		return err
	}
	if !enabled {
		return nil
	}
	if executor, ok := n.wrapped.(Executor); ok {
		return executor.Execute(wire)
	}
	return nil
}
