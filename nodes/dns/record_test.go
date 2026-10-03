package dns

import (
	"testing"

	"github.com/ntnn/tensile/pkg/diff"
	"github.com/stretchr/testify/assert"
)

func TestRecord_Equal(t *testing.T) {
	t.Parallel()

	base := Record{
		SubName: "www",
		Type:    "A",
		Records: []string{"1.2.3.4", "5.6.7.8"},
		TTL:     300,
	}

	cases := map[string]struct {
		record   Record
		other    Record
		expected bool
	}{
		"identical": {
			record:   base,
			other:    base,
			expected: true,
		},
		"same entries different order": {
			record:   base,
			other:    Record{SubName: "www", Type: "A", Records: []string{"5.6.7.8", "1.2.3.4"}, TTL: 300},
			expected: true,
		},
		"case insensitive subname and type": {
			record:   base,
			other:    Record{SubName: "WWW", Type: "a", Records: []string{"1.2.3.4", "5.6.7.8"}, TTL: 300},
			expected: true,
		},
		"different subname": {
			record:   base,
			other:    Record{SubName: "mail", Type: "A", Records: []string{"1.2.3.4", "5.6.7.8"}, TTL: 300},
			expected: false,
		},
		"different type": {
			record:   base,
			other:    Record{SubName: "www", Type: "AAAA", Records: []string{"1.2.3.4", "5.6.7.8"}, TTL: 300},
			expected: false,
		},
		"different content same length": {
			record:   base,
			other:    Record{SubName: "www", Type: "A", Records: []string{"1.2.3.4", "9.9.9.9"}, TTL: 300},
			expected: false,
		},
		"different length": {
			record:   base,
			other:    Record{SubName: "www", Type: "A", Records: []string{"1.2.3.4"}, TTL: 300},
			expected: false,
		},
		"different ttl both managed": {
			record:   base,
			other:    Record{SubName: "www", Type: "A", Records: []string{"1.2.3.4", "5.6.7.8"}, TTL: 600},
			expected: false,
		},
		"ttl zero ignores current ttl": {
			record:   Record{SubName: "www", Type: "A", Records: []string{"1.2.3.4", "5.6.7.8"}},
			other:    base,
			expected: true,
		},
		"ttl set current ttl zero": {
			record:   base,
			other:    Record{SubName: "www", Type: "A", Records: []string{"1.2.3.4", "5.6.7.8"}},
			expected: false,
		},
	}

	for title, cas := range cases {
		t.Run(title, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, cas.expected, cas.record.Equal(cas.other))
		})
	}
}

func TestRecord_Diff(t *testing.T) {
	t.Parallel()

	cases := map[string]struct {
		record   Record
		other    Record
		expected []*diff.FieldChange
	}{
		"identical": {
			record:   Record{SubName: "www", Type: "A", Records: []string{"1.2.3.4"}, TTL: 300},
			other:    Record{SubName: "www", Type: "A", Records: []string{"1.2.3.4"}, TTL: 300},
			expected: []*diff.FieldChange{},
		},
		"entry added": {
			record: Record{SubName: "www", Type: "A", Records: []string{"1.2.3.4", "5.6.7.8"}},
			other:  Record{SubName: "www", Type: "A", Records: []string{"1.2.3.4"}},
			expected: []*diff.FieldChange{
				{Field: "www A", Old: diff.Absent, New: "5.6.7.8"},
			},
		},
		"entry removed": {
			record: Record{SubName: "www", Type: "A", Records: []string{"1.2.3.4"}},
			other:  Record{SubName: "www", Type: "A", Records: []string{"1.2.3.4", "5.6.7.8"}},
			expected: []*diff.FieldChange{
				{Field: "www A", Old: "5.6.7.8", New: diff.Absent},
			},
		},
		"entry replaced": {
			record: Record{SubName: "www", Type: "A", Records: []string{"9.9.9.9"}},
			other:  Record{SubName: "www", Type: "A", Records: []string{"1.2.3.4"}},
			expected: []*diff.FieldChange{
				{Field: "www A", Old: "1.2.3.4", New: diff.Absent},
				{Field: "www A", Old: diff.Absent, New: "9.9.9.9"},
			},
		},
		"ttl changed": {
			record: Record{SubName: "www", Type: "A", Records: []string{"1.2.3.4"}, TTL: 600},
			other:  Record{SubName: "www", Type: "A", Records: []string{"1.2.3.4"}, TTL: 300},
			expected: []*diff.FieldChange{
				{Field: "www A", Old: "1.2.3.4 (TTL 300)", New: "1.2.3.4 (TTL 600)"},
			},
		},
		"unmanaged ttl ignored": {
			record:   Record{SubName: "www", Type: "A", Records: []string{"1.2.3.4"}},
			other:    Record{SubName: "www", Type: "A", Records: []string{"1.2.3.4"}, TTL: 300},
			expected: []*diff.FieldChange{},
		},
		"ttl set over unmanaged": {
			record: Record{SubName: "www", Type: "A", Records: []string{"1.2.3.4"}, TTL: 300},
			other:  Record{SubName: "www", Type: "A", Records: []string{"1.2.3.4"}},
			expected: []*diff.FieldChange{
				{Field: "www A", Old: "1.2.3.4", New: "1.2.3.4 (TTL 300)"},
			},
		},
		"negative ttl unmanaged": {
			record:   Record{SubName: "www", Type: "A", Records: []string{"1.2.3.4"}, TTL: -1},
			other:    Record{SubName: "www", Type: "A", Records: []string{"1.2.3.4"}, TTL: 300},
			expected: []*diff.FieldChange{},
		},
		"negative ttl omitted in added line": {
			record: Record{SubName: "www", Type: "A", Records: []string{"1.2.3.4"}, TTL: -1},
			other:  Record{},
			expected: []*diff.FieldChange{
				{Field: "www A", Old: diff.Absent, New: "1.2.3.4"},
			},
		},
		"create from zero value": {
			record: Record{SubName: "www", Type: "A", Records: []string{"1.2.3.4", "5.6.7.8"}, TTL: 300},
			other:  Record{},
			expected: []*diff.FieldChange{
				{Field: "www A", Old: diff.Absent, New: "1.2.3.4 (TTL 300)"},
				{Field: "www A", Old: diff.Absent, New: "5.6.7.8 (TTL 300)"},
			},
		},
	}

	for title, cas := range cases {
		t.Run(title, func(t *testing.T) {
			t.Parallel()
			assert.ElementsMatch(t, cas.expected, cas.record.Diff(cas.other))
		})
	}
}
