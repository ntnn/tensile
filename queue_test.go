package tensile

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
)

type testNode struct {
	Name string
}

func (t testNode) Identity() (Shape, string) {
	return Noop, t.Name
}

func TestQueue_Add(t *testing.T) {
	cases := map[string]struct {
		input       []Identitier
		expectedErr error
	}{
		"add single element": {
			input: []Identitier{
				&testNode{Name: "/example"},
			},
		},
		"add multiple elements": {
			input: []Identitier{
				&testNode{Name: "/example"},
				&testNode{Name: "/example1"},
				&testNode{Name: "/example2"},
			},
		},
		"two elements with same identity fails": {
			input: []Identitier{
				&testNode{Name: "/example"},
				&testNode{Name: "/example"},
			},
			expectedErr: fmt.Errorf("same identity already registered"),
		},
	}

	for title, cas := range cases {
		t.Run(title, func(t *testing.T) {
			assert.Equal(t, cas.expectedErr, NewQueue().Add(cas.input...))
		})
	}
}
