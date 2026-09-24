package diff

import (
	"log/slog"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestFieldChanges_String(t *testing.T) {
	t.Parallel()

	d := FieldChanges{Fields: []FieldChange{
		{Field: "mode", Old: "0644", New: "0600"},
		{Field: "owner", Old: Absent, New: "root"},
	}}
	assert.Equal(t, "mode: 0644 -> 0600\nowner: (absent) -> root", d.String())
	assert.Empty(t, FieldChanges{}.String())
}

func TestFieldChanges_LogValue(t *testing.T) {
	t.Parallel()

	d := FieldChanges{Fields: []FieldChange{
		{Field: "mode", Old: "0644", New: "0600"},
	}}
	value := d.LogValue()
	assert.Equal(t, slog.KindGroup, value.Kind())
	group := value.Group()
	assert.Len(t, group, 1)
	assert.Equal(t, "mode", group[0].Key)
	assert.Equal(t, "0644 -> 0600", group[0].Value.String())
}
