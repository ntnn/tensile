package tensile

import (
	"os"
	"testing"

	"github.com/ntnn/tensile/facts"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTemplateString(t *testing.T) {
	f, err := facts.New()
	require.Nil(t, err)

	// empty template / nil customdata yields no error
	s, err := TemplateString(f, "", nil)
	assert.Nil(t, err)
	assert.Equal(t, "", s)

	// non-empty template / nil customdata
	s, err = TemplateString(f, "{{.Custom.Test}}", nil)
	assert.Nil(t, err)
	assert.Equal(t, "<no value>", s)

	// value from customdata
	s, err = TemplateString(f, "{{.Custom.test}}", map[string]any{"test": "Hello, world!"})
	assert.Nil(t, err)
	assert.Equal(t, "Hello, world!", s)

	// value from facts
	hostname, err := os.Hostname()
	if assert.Nil(t, err) && assert.NotEqual(t, "", hostname) {
		s, err = TemplateString(f, "{{.Facts.Hostname}}", nil)
		assert.Nil(t, err)
		assert.Equal(t, hostname, s)
	}
}
