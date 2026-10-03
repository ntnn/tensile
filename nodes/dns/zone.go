package dns

import (
	"errors"
	"fmt"
	"strings"

	"github.com/ntnn/tensile"
	"github.com/ntnn/tensile/pkg/diff"
)

var (
	_ tensile.Identifier = (*Zone)(nil)
	_ tensile.Validator  = (*Zone)(nil)
	_ tensile.Conflictor = (*Zone)(nil)
	_ tensile.Executor   = (*Zone)(nil)
)

// Zone manages records in Domain using Service.
type Zone struct {
	Domain  string
	Service Service
	Records []Record
	// IgnoreSubNames excludes externally managed subnames from pruning.
	// E.g. records managed by dynamic DNS services.
	IgnoreSubNames []string
}

// Identity implements [tensile.Identifier].
func (zone *Zone) Identity() tensile.Identity {
	return tensile.AsIdentity("dnsZone", "domain", strings.ToLower(zone.Domain))
}

// Validate implements [tensile.Validator].
func (zone *Zone) Validate(_ tensile.Wire) error {
	if zone.Domain == "" {
		return errors.New("domain is required")
	}
	if zone.Service == nil {
		return errors.New("service is required")
	}
	seen := map[tensile.Identity]struct{}{}
	for _, record := range zone.Records {
		if err := record.Validate(); err != nil {
			return fmt.Errorf("record %q: %w", record, err)
		}
		recordID := RecordIdentity(zone.Domain, record.SubName, record.Type)
		if _, ok := seen[recordID]; ok {
			return fmt.Errorf("record %q: declared twice", record)
		}
		seen[recordID] = struct{}{}
	}
	return nil
}

// Conflicts implements [tensile.Conflictor].
func (zone *Zone) Conflicts() ([]tensile.Identity, error) {
	identities := make([]tensile.Identity, len(zone.Records))
	for i, record := range zone.Records {
		identities[i] = RecordIdentity(zone.Domain, record.SubName, record.Type)
	}
	return identities, nil
}

// NeedsExecution implements [tensile.Executor].
func (zone *Zone) NeedsExecution(wire tensile.Wire) (bool, tensile.Diff, error) {
	create, update, remove, err := zone.changes(wire)
	if err != nil {
		return false, nil, err
	}
	if len(create) == 0 && len(update) == 0 && len(remove) == 0 {
		return false, nil, nil
	}

	changes := []*diff.FieldChange{}
	for _, record := range create {
		changes = append(changes, record.Diff(Record{})...)
	}
	for _, upd := range update {
		changes = append(changes, upd.desired.Diff(upd.current)...)
	}
	for _, record := range remove {
		empty := Record{
			SubName: record.SubName,
			Type:    record.Type,
		}
		changes = append(changes, empty.Diff(record)...)
	}
	return true, diff.NewFieldChanges(changes...), nil
}

// Execute implements [tensile.Executor].
func (zone *Zone) Execute(wire tensile.Wire) (tensile.Diff, error) {
	create, update, remove, err := zone.changes(wire)
	if err != nil {
		return nil, err
	}

	if len(create) > 0 {
		if err := zone.Service.Create(wire.Context(), zone.Domain, create); err != nil {
			return nil, fmt.Errorf("creating records: %w", err)
		}
	}

	if len(update) > 0 {
		records := make([]Record, len(update))
		for i, upd := range update {
			records[i] = upd.desired
		}
		if err := zone.Service.Update(wire.Context(), zone.Domain, records); err != nil {
			return nil, fmt.Errorf("updating records: %w", err)
		}
	}

	if len(remove) > 0 {
		if err := zone.Service.Delete(wire.Context(), zone.Domain, remove); err != nil {
			return nil, fmt.Errorf("deleting records: %w", err)
		}
	}

	return nil, nil //nolint:nilnil // nil Diff is valid
}

// recordUpdate pairs a differing record with its current state for diffing.
type recordUpdate struct {
	current Record
	desired Record
}

func (zone *Zone) changes(wire tensile.Wire) ([]Record, []recordUpdate, []Record, error) {
	existingList, err := zone.Service.List(wire.Context(), zone.Domain)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("listing records: %w", err)
	}

	merged := MergeRecords(existingList)
	existing := map[tensile.Identity]Record{}
	for _, record := range merged {
		existing[RecordIdentity(zone.Domain, record.SubName, record.Type)] = record
	}

	create := []Record{}
	update := []recordUpdate{}
	managed := map[tensile.Identity]struct{}{}
	for _, record := range zone.Records {
		identity := RecordIdentity(zone.Domain, record.SubName, record.Type)
		managed[identity] = struct{}{}
		current, ok := existing[identity]
		if !ok {
			create = append(create, record)
			continue
		}
		if !record.Equal(current) {
			update = append(update, recordUpdate{current: current, desired: record})
		}
	}

	ignore := map[string]struct{}{}
	for _, subname := range zone.IgnoreSubNames {
		ignore[strings.ToLower(subname)] = struct{}{}
	}

	remove := []Record{}
	for _, record := range merged {
		if _, ok := ignore[strings.ToLower(record.SubName)]; ok {
			continue
		}
		identity := RecordIdentity(zone.Domain, record.SubName, record.Type)
		if _, ok := managed[identity]; ok {
			continue
		}
		remove = append(remove, record)
	}

	return create, update, remove, nil
}
