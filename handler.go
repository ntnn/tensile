package tensile

// Handler is a [Node] that is only executed when at least one of the
// nodes notifying it was executed.
type Handler struct {
	Node
}

// NewHandler takes any value and transforms it into a [Handler].
func NewHandler(input any) (*Handler, error) {
	if handler, ok := input.(*Handler); ok {
		return handler, nil
	}

	node, err := NewNode(input)
	if err != nil {
		return nil, err
	}
	return &Handler{Node: *node}, nil
}
