package tensile

import "fmt"

type Shaper interface {
	Shape() Shape
}

type Identifier interface {
	Identifier() string
}

type Node interface {
	Shaper
	Identifier
}

type NodeWrapper struct {
	Node Node
}

func NodeWrap(node Node) *NodeWrapper {
	return &NodeWrapper{
		Node: node,
	}
}

func (nw NodeWrapper) Identity() (Shape, string) {
	return nw.Node.Shape(), nw.Node.Identifier()
}

func (nw NodeWrapper) String() string {
	return fmt.Sprintf("%s[%s]", nw.Node.Shape(), nw.Node.Identifier())
}

// Validator validates an element when adding it to a queue.
// The error will be returned by the queue if validation fails.
//
// This can be used to e.g. validate that all required options are set
// or to set internal values.
type Validator interface {
	Validate() error
}

func (nw NodeWrapper) Validate() error {
	if validator, ok := nw.Node.(Validator); ok {
		return validator.Validate()
	}
	return nil
}

// IsCollisioner is used to detect wether two nodes with the same
// identity cause a collision.
type IsCollisioner interface {
	// IsCollision receives another node and should return an error if they
	// are colliding.
	// See the Package node for an example.
	IsCollision(other Node) error
}

func (nw NodeWrapper) IsCollision(other NodeWrapper) error {
	if isCollisioner, ok := nw.Node.(IsCollisioner); ok {
		return isCollisioner.IsCollision(other.Node)
	}

	// TODO default implementation
	return nil
}

type NeedsExecutioner interface {
	NeedsExecution(Context) (bool, error)
}

func (nw NodeWrapper) NeedsExecution(ctx Context) (bool, error) {
	if needsExecutioner, ok := nw.Node.(NeedsExecutioner); ok {
		return needsExecutioner.NeedsExecution(ctx)
	}
	return true, nil
}

// Executor executes the node.
type Executor interface {
	// Execute executes the node.
	// The return value can be a struct to be utilized by other
	// elements.
	Execute(Context) (any, error)
}

func (nw NodeWrapper) Execute(ctx Context) (any, error) {
	if executor, ok := nw.Node.(Executor); ok {
		return executor.Execute(ctx)
	}
	return nil, nil
}
