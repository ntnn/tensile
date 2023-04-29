package tensile

import (
	"errors"
)

type Node interface {
	Shape() Shape
	Identifier() string
}

var _ Node = (*NodeWrapper)(nil)

// NodeWrapper wraps around nodes to provide some common functionality.
type NodeWrapper struct {
	Node Node

	Before, After []string
}

func NodeWrap(node Node) NodeWrapper {
	// Ensure a node wrapper is not wrapped
	if nw, ok := node.(*NodeWrapper); ok {
		return *nw
	}

	return NodeWrapper{
		Node: node,
	}
}

func (nw NodeWrapper) Shape() Shape {
	return nw.Node.Shape()
}

func (nw NodeWrapper) Identifier() string {
	return nw.Node.Identifier()
}

func (nw NodeWrapper) Identity() (Shape, string) {
	return nw.Node.Shape(), nw.Node.Identifier()
}

func (nw NodeWrapper) String() string {
	return FormatIdentity(nw.Node.Shape(), nw.Node.Identifier())
}

// Validator validates an element when adding it to a queue.
// The error will be returned by the queue if validation fails.
//
// This can be used to e.g. validate that all required options are set
// or to set internal values.
type Validator interface {
	Validate() error
}

var ErrNodeIsNil = errors.New("tensile: node in wrapper is nil")

func (nw NodeWrapper) Validate() error {
	if nw.Node == nil {
		return ErrNodeIsNil
	}
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

	return ErrIsCollisionerNotImplemented
}

type AfterNoder interface {
	// AfterNodes lists nodes after which this node must be executed if they exist.
	AfterNodes() []string
}

func (nw NodeWrapper) AfterNodes() []string {
	after := nw.After
	if afterNoder, ok := nw.Node.(AfterNoder); ok {
		after = append(after, afterNoder.AfterNodes()...)
	}
	return after
}

type BeforeNoder interface {
	// BeforeNodes lists nodes before which this node must be executed if they exist.
	BeforeNodes() []string
}

func (nw NodeWrapper) BeforeNodes() []string {
	before := nw.Before
	if beforeNoder, ok := nw.Node.(BeforeNoder); ok {
		before = append(before, beforeNoder.BeforeNodes()...)
	}
	return before
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

type Executor interface {
	Execute(Context) (any, error)
}

func (nw NodeWrapper) Execute(ctx Context) (any, error) {
	if executor, ok := nw.Node.(Executor); ok {
		return executor.Execute(ctx)
	}
	return nil, nil
}
