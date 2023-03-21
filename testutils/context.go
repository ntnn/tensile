package testutils

import (
	"testing"

	"github.com/ntnn/tensile"
	"github.com/ntnn/tensile/facts"
	"github.com/stretchr/testify/require"
)

func Context(t *testing.T) tensile.Context {
	ctx, err := tensile.NewContext(nil, nil, facts.Facts{})
	require.Nil(t, err)
	return ctx
}
