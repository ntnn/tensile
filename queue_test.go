package gorrect

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestQueue_Add(t *testing.T) {
	cases := map[string]struct {
		input       []Identitier
		expectedErr error
	}{
		"add single element": {
			input: []Identitier{
				&File{Target: "/example"},
			},
		},
		"add multiple elements": {
			input: []Identitier{
				&File{Target: "/example"},
				&File{Target: "/example1"},
				&File{Target: "/example2"},
			},
		},
		"two elements with same identity fails": {
			input: []Identitier{
				&File{Target: "/example"},
				&File{Target: "/example"},
			},
			expectedErr: fmt.Errorf("same identity already registered"),
		},
		"two noop elements with same identity does not fail": {
			input: []Identitier{
				&Log{Message: "Hello, world!"},
				&Log{Message: "Hello, world!"},
			},
		},
	}

	for title, cas := range cases {
		t.Run(title, func(t *testing.T) {
			assert.Equal(t, cas.expectedErr, NewQueue().Add(cas.input...))
		})
	}
}
