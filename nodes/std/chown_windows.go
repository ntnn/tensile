//go:build windows

package std

import (
	"github.com/ntnn/tensile/pkg/diff"
)

func chownNeedsExecution(_ string, _, _ string) (bool, *diff.FieldChange, *diff.FieldChange, error) {
	return false, nil, nil, nil
}

func chownApply(_ string, _, _ string) error {
	return nil
}
