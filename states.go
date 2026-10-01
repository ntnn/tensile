package tensile

import "fmt"

// State may be used by nodes to declare the desired state of the
// resource it manages.
// It is purely a convention.
// Nodes may also define their own type to express state.
type State string

const (
	// Present is when the resource the node manages should be present.
	// For most nodes using [State] it should be the default.
	Present State = "present"
	// Absent is when the resource the node manages should not be present.
	Absent State = "absent"
	// ReadOnly may be supported by nodes that support only reporting on the managed resource.
	ReadOnly State = "read-only"
)

// OrDefault returns the state or [Present] if it is empty string.
func (s State) OrDefault() State {
	if s == "" {
		return Present
	}
	return s
}

// Valid returns an error if s isn't a defined [State] or empty string.
func (s State) Valid() error {
	switch s {
	case "", Present, Absent, ReadOnly:
		return nil
	default:
		return fmt.Errorf("%q is not a defined %T", s, s)
	}
}
