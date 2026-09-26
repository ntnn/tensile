package tensile

// Handler is a [Node] that is only executed when at least one of the
// nodes notifying it was executed.
type Handler struct {
	Node
}

// NewHandler wraps an [Identifier] into a [Handler].
func NewHandler(input Identifier) *Handler {
	if handler, ok := input.(*Handler); ok {
		return handler
	}
	node := NewNode(input)
	node.isHandler = true
	return &Handler{Node: *node}
}
