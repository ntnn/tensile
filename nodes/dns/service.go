package dns

import (
	"context"
	"maps"
	"slices"
	"strings"

	"github.com/ntnn/tensile"
)

// Service is the expected interface to interface with a DNS provider.
type Service interface {
	// List returns all records of domain.
	List(ctx context.Context, domain string) ([]Record, error)
	// Create creates records in domain.
	// If a record has a TTL of zero or less the implementation should
	// fall back to the default the DNS provider expects.
	Create(ctx context.Context, domain string, records []Record) error
	// Update replaces existing records in domain.
	// If a record has a TTL of zero or less the implementation should
	// fall back to the default the DNS provider expects.
	Update(ctx context.Context, domain string, records []Record) error
	// Delete removes records from domain.
	Delete(ctx context.Context, domain string, records []Record) error
}

// MergeRecords merges records sharing a subname and type.
// The lowest TTL greater than 0 wins.
// The returned records are sorted.
func MergeRecords(records []Record) []Record {
	merged := map[tensile.Identity]Record{}

	for _, record := range records {
		identity := RecordIdentity("", record.SubName, record.Type)

		current, ok := merged[identity]
		if !ok {
			record.Records = slices.Clone(record.Records)
			slices.Sort(record.Records)
			record.Records = slices.Compact(record.Records)
			merged[identity] = record
			continue
		}

		current.Records = slices.Concat(current.Records, record.Records)
		slices.Sort(current.Records)
		current.Records = slices.Compact(current.Records)

		if record.TTL > 0 && record.TTL < current.TTL {
			current.TTL = record.TTL
		}
		if record.TTL > 0 && current.TTL <= 0 {
			current.TTL = record.TTL
		}

		merged[identity] = current
	}

	result := slices.Collect(maps.Values(merged))
	slices.SortFunc(result, compareRecords)
	return result
}

func compareRecords(a, b Record) int {
	if c := strings.Compare(strings.ToLower(a.SubName), strings.ToLower(b.SubName)); c != 0 {
		return c
	}
	return strings.Compare(strings.ToUpper(a.Type), strings.ToUpper(b.Type))
}

// SplitRecords splits each [Record] into one [Record] per [Record.Records].
func SplitRecords(records []Record) []Record {
	result := []Record{}
	for _, record := range records {
		for _, entry := range record.Records {
			split := record
			split.Records = []string{entry}
			result = append(result, split)
		}
	}
	return result
}
