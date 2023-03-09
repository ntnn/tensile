package tensile

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func Dir_Validate_setBaseName(t *testing.T) {
	d := Dir{
		Target: "/a/t/b",
	}

	if err := d.Validate(); err != nil {
		t.Errorf("unexepected error in validation: %v", err)
		return
	}

	assert.Equal(t, d.Target, d.Base.Name)
}

func Dir_Validate_customBaseName(t *testing.T) {
	d := Dir{
		Base: Base{
			Name: "a random name",
		},
		Target: "/a/t/b",
	}

	if err := d.Validate(); err != nil {
		t.Errorf("unexepected error in validation: %v", err)
		return
	}

	assert.NotEqual(t, d.Target, d.Base.Name)
}
