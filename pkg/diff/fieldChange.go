package diff

import (
	"log/slog"
	"strings"

	"github.com/ntnn/tensile"
)

var _ tensile.Diff = FieldChanges{}

// Absent marks a nonexistent value on one side of a change.
const Absent = "(absent)"

// FieldChange is a single field-level change from Old to New.
type FieldChange struct {
	Field, Old, New string
}

// FieldChanges is a list of field-level changes.
type FieldChanges struct {
	Fields []FieldChange
}

// String renders one "field: old -> new" line per change.
func (d FieldChanges) String() string {
	var b strings.Builder
	for i, f := range d.Fields {
		if i > 0 {
			b.WriteByte('\n')
		}
		b.WriteString(f.Field)
		b.WriteString(": ")
		b.WriteString(f.Old)
		b.WriteString(" -> ")
		b.WriteString(f.New)
	}
	return b.String()
}

// LogValue renders the changes as a group, one attr per field.
func (d FieldChanges) LogValue() slog.Value {
	attrs := make([]slog.Attr, len(d.Fields))
	for i, f := range d.Fields {
		attrs[i] = slog.String(f.Field, f.Old+" -> "+f.New)
	}
	return slog.GroupValue(attrs...)
}
