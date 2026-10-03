package dns

import (
	"errors"
	"fmt"
	"slices"
	"strings"

	"github.com/ntnn/tensile"
	"github.com/ntnn/tensile/pkg/diff"
)

// Record is a set of DNS records with the same subname and type.
type Record struct {
	SubName string
	Type    string
	Records []string
	// A TTL of zero or less means unmanaged.
	// On Creation the [Service] may decide which TTL to apply, but
	// should fall back to the DNS providers TTL.
	// On Update an unmanaged TTL means a previously set TTL will
	// persist.
	TTL int
}

// RecordIdentity returns the identity of a record in a domain.
func RecordIdentity(domain, subname, rtype string) tensile.Identity {
	return tensile.AsIdentity("dnsRecord",
		"domain", strings.ToLower(domain),
		"subname", strings.ToLower(subname),
		"type", strings.ToUpper(rtype),
	)
}

// Validate validates a Record.
func (record Record) Validate() error {
	if record.Type == "" {
		return errors.New("type is required")
	}
	if len(record.Records) == 0 {
		return errors.New("records are required")
	}
	return nil
}

func (record Record) String() string {
	subname := record.SubName
	if subname == "" {
		subname = "."
	}
	return fmt.Sprintf("%s %s", subname, record.Type)
}

// Diff returns the field changes from other to record.
func (record Record) Diff(other Record) []*diff.FieldChange {
	field := record.String()

	current := slices.Clone(other.Records)
	desired := slices.Clone(record.Records)
	slices.Sort(current)
	slices.Sort(desired)

	changes := []*diff.FieldChange{}
	// deleted records
	for _, entry := range current {
		if !slices.Contains(desired, entry) {
			changes = append(changes, &diff.FieldChange{
				Field: field,
				Old:   line(entry, other.TTL),
				New:   diff.Absent,
			})
		}
	}
	for _, entry := range desired {
		// added records
		if !slices.Contains(current, entry) {
			changes = append(changes, &diff.FieldChange{
				Field: field,
				Old:   diff.Absent,
				New:   line(entry, record.TTL),
			})
			continue
		}
		// changed records - only TTL
		if record.TTL > 0 && record.TTL != other.TTL {
			changes = append(changes, &diff.FieldChange{
				Field: field,
				Old:   line(entry, other.TTL),
				New:   line(entry, record.TTL),
			})
		}
	}
	return changes
}

// line renders a record entry with its TTL.
// An unmanaged TTL is omitted.
func line(entry string, ttl int) string {
	if ttl <= 0 {
		return entry
	}
	return fmt.Sprintf("%s (TTL %d)", entry, ttl)
}

// Equal returns true if the two records have the same properties.
// SubName and Type are compared case-insensitively,
// TTL is only considered if record's TTL greater than 0.
// Records are compared after sorting.
func (record Record) Equal(other Record) bool {
	if !strings.EqualFold(record.SubName, other.SubName) ||
		!strings.EqualFold(record.Type, other.Type) ||
		len(record.Records) != len(other.Records) {
		return false
	}
	if record.TTL > 0 && record.TTL != other.TTL {
		return false
	}
	a := slices.Clone(record.Records)
	b := slices.Clone(other.Records)
	slices.Sort(a)
	slices.Sort(b)
	return slices.Equal(a, b)
}
