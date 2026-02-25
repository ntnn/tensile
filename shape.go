package tensile

import "fmt"

type Shape string

func (s Shape) AsRef(input any) NodeRef {
	return NodeRef{Shape: s, Value: input}
}

func (s Shape) AsRefs(inputs ...any) []NodeRef {
	ret := make([]NodeRef, len(inputs))
	for i, input := range inputs {
		ret[i] = s.AsRef(input)
	}
	return ret
}

type NodeRef struct {
	Shape Shape
	Value any
}

func (n NodeRef) String() string {
	return fmt.Sprintf("%s[%v]", n.Shape, n.Value)
}
