package tensile

import "fmt"

// Ref the type of a node reference.
type Ref string

// To creates a [NodeRef] with the given value.
func (ref Ref) To(input any) NodeRef {
	return NodeRef{Ref: ref, Value: input}
}

// ToMany is equivalent for [Ref.To] but for many values.
func (ref Ref) ToMany(inputs ...any) []NodeRef {
	ret := make([]NodeRef, len(inputs))
	for i, input := range inputs {
		ret[i] = ref.To(input)
	}
	return ret
}

// NodeRef is a reference to a node.
type NodeRef struct {
	Ref   Ref
	Value any
}

// String implements the [fmt.Stringer] interface.
func (n NodeRef) String() string {
	return fmt.Sprintf("%s<%v>", n.Ref, n.Value)
}
