package tensile

import "fmt"

type Ref string

func (ref Ref) To(input any) NodeRef {
	return NodeRef{Ref: ref, Value: input}
}

func (ref Ref) ToMany(inputs ...any) []NodeRef {
	ret := make([]NodeRef, len(inputs))
	for i, input := range inputs {
		ret[i] = ref.To(input)
	}
	return ret
}

type NodeRef struct {
	Ref   Ref
	Value any
}

func (n NodeRef) String() string {
	return fmt.Sprintf("%s[%v]", n.Ref, n.Value)
}
