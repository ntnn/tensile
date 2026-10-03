package dns

import (
	"context"
	"errors"
	"slices"
	"testing"

	"github.com/ntnn/tensile"
	"github.com/ntnn/tensile/pkg/diff"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var _ Service = (*mockService)(nil)

// mockService records mutations and serves records for List.
type mockService struct {
	records []Record

	listErr   error
	createErr error
	updateErr error
	deleteErr error

	created []Record
	updated []Record
	deleted []Record
}

func (m *mockService) List(_ context.Context, _ string) ([]Record, error) {
	if m.listErr != nil {
		return nil, m.listErr
	}
	return slices.Clone(m.records), nil
}

func (m *mockService) Create(_ context.Context, _ string, records []Record) error {
	if m.createErr != nil {
		return m.createErr
	}
	m.created = append(m.created, records...)
	return nil
}

func (m *mockService) Update(_ context.Context, _ string, records []Record) error {
	if m.updateErr != nil {
		return m.updateErr
	}
	m.updated = append(m.updated, records...)
	return nil
}

func (m *mockService) Delete(_ context.Context, _ string, records []Record) error {
	if m.deleteErr != nil {
		return m.deleteErr
	}
	m.deleted = append(m.deleted, records...)
	return nil
}

// mutations returns true if any mutating method was called.
func (m *mockService) mutations() bool {
	return len(m.created)+len(m.updated)+len(m.deleted) > 0
}

func TestZone_Validate(t *testing.T) {
	t.Parallel()

	cases := map[string]struct {
		zone        Zone
		expectedErr string
	}{
		"valid": {
			zone: Zone{
				Domain:  "example.org",
				Service: &mockService{},
				Records: []Record{
					{SubName: "www", Type: "A", Records: []string{"1.2.3.4"}},
				},
			},
		},
		"empty domain": {
			zone:        Zone{Service: &mockService{}},
			expectedErr: "domain is required",
		},
		"nil service": {
			zone:        Zone{Domain: "example.org"},
			expectedErr: "service is required",
		},
		"invalid record": {
			zone: Zone{
				Domain:  "example.org",
				Service: &mockService{},
				Records: []Record{
					{SubName: "www", Type: "A"},
				},
			},
			expectedErr: "records are required",
		},
		"duplicate record entry": {
			zone: Zone{
				Domain:  "example.org",
				Service: &mockService{},
				Records: []Record{
					{SubName: "www", Type: "A", Records: []string{"1.2.3.4", "1.2.3.4"}},
				},
			},
			expectedErr: `duplicate entry "1.2.3.4"`,
		},
		"duplicate record": {
			zone: Zone{
				Domain:  "example.org",
				Service: &mockService{},
				Records: []Record{
					{SubName: "www", Type: "A", Records: []string{"1.2.3.4"}},
					{SubName: "www", Type: "A", Records: []string{"5.6.7.8"}},
				},
			},
			expectedErr: "declared twice",
		},
		"duplicate record case variant": {
			zone: Zone{
				Domain:  "example.org",
				Service: &mockService{},
				Records: []Record{
					{SubName: "www", Type: "A", Records: []string{"1.2.3.4"}},
					{SubName: "WWW", Type: "a", Records: []string{"5.6.7.8"}},
				},
			},
			expectedErr: "declared twice",
		},
		"same subname different type": {
			zone: Zone{
				Domain:  "example.org",
				Service: &mockService{},
				Records: []Record{
					{SubName: "www", Type: "A", Records: []string{"1.2.3.4"}},
					{SubName: "www", Type: "AAAA", Records: []string{"::1"}},
				},
			},
		},
	}

	for title, cas := range cases {
		t.Run(title, func(t *testing.T) {
			t.Parallel()
			err := cas.zone.Validate(nil)
			if cas.expectedErr == "" {
				require.NoError(t, err)
				return
			}
			require.ErrorContains(t, err, cas.expectedErr)
		})
	}
}

// zoneCase drives both NeedsExecution and Execute, which derive
// from the same change calculation.
type zoneCase struct {
	records        []Record
	ignoreSubNames []string
	remote         []Record

	expectedNeeds   bool
	expectedFields  []diff.FieldChange
	expectedCreated []Record
	expectedUpdated []Record
	expectedDeleted []Record
}

func zoneCases() map[string]zoneCase {
	return map[string]zoneCase{
		"no records no remote": {},
		"create missing record": {
			records: []Record{
				{SubName: "www", Type: "A", Records: []string{"1.2.3.4"}, TTL: 300},
			},
			expectedNeeds: true,
			expectedFields: []diff.FieldChange{
				{Field: "www A", Old: diff.Absent, New: "1.2.3.4 (TTL 300)"},
			},
			expectedCreated: []Record{
				{SubName: "www", Type: "A", Records: []string{"1.2.3.4"}, TTL: 300},
			},
		},
		"create with unmanaged ttl": {
			records: []Record{
				{SubName: "www", Type: "A", Records: []string{"1.2.3.4"}},
			},
			expectedNeeds: true,
			expectedFields: []diff.FieldChange{
				{Field: "www A", Old: diff.Absent, New: "1.2.3.4"},
			},
			expectedCreated: []Record{
				{SubName: "www", Type: "A", Records: []string{"1.2.3.4"}},
			},
		},
		"update differing content": {
			records: []Record{
				{SubName: "www", Type: "A", Records: []string{"1.2.3.4"}, TTL: 300},
			},
			remote: []Record{
				{SubName: "www", Type: "A", Records: []string{"5.6.7.8"}, TTL: 300},
			},
			expectedNeeds: true,
			expectedFields: []diff.FieldChange{
				{Field: "www A", Old: "5.6.7.8 (TTL 300)", New: diff.Absent},
				{Field: "www A", Old: diff.Absent, New: "1.2.3.4 (TTL 300)"},
			},
			expectedUpdated: []Record{
				{SubName: "www", Type: "A", Records: []string{"1.2.3.4"}, TTL: 300},
			},
		},
		"update differing ttl": {
			records: []Record{
				{SubName: "www", Type: "A", Records: []string{"1.2.3.4"}, TTL: 600},
			},
			remote: []Record{
				{SubName: "www", Type: "A", Records: []string{"1.2.3.4"}, TTL: 300},
			},
			expectedNeeds: true,
			expectedFields: []diff.FieldChange{
				{Field: "www A", Old: "1.2.3.4 (TTL 300)", New: "1.2.3.4 (TTL 600)"},
			},
			expectedUpdated: []Record{
				{SubName: "www", Type: "A", Records: []string{"1.2.3.4"}, TTL: 600},
			},
		},
		"ttl zero matches any remote ttl": {
			records: []Record{
				{SubName: "www", Type: "A", Records: []string{"1.2.3.4"}},
			},
			remote: []Record{
				{SubName: "www", Type: "A", Records: []string{"1.2.3.4"}, TTL: 300},
			},
		},
		"case variant remote record matches": {
			records: []Record{
				{SubName: "www", Type: "A", Records: []string{"1.2.3.4"}, TTL: 300},
			},
			remote: []Record{
				{SubName: "WWW", Type: "a", Records: []string{"1.2.3.4"}, TTL: 300},
			},
		},
		"case variant ignored subname kept": {
			ignoreSubNames: []string{"ddns"},
			remote: []Record{
				{SubName: "DDNS", Type: "A", Records: []string{"9.9.9.9"}, TTL: 60},
			},
		},
		"split remote record set merged": {
			records: []Record{
				{SubName: "www", Type: "A", Records: []string{"1.2.3.4", "5.6.7.8"}, TTL: 300},
			},
			remote: []Record{
				{SubName: "www", Type: "A", Records: []string{"1.2.3.4"}, TTL: 300},
				{SubName: "www", Type: "A", Records: []string{"5.6.7.8"}, TTL: 300},
			},
		},
		"delete unmanaged record": {
			records: []Record{
				{SubName: "www", Type: "A", Records: []string{"1.2.3.4"}, TTL: 300},
			},
			remote: []Record{
				{SubName: "www", Type: "A", Records: []string{"1.2.3.4"}, TTL: 300},
				{SubName: "old", Type: "A", Records: []string{"9.9.9.9"}, TTL: 300},
			},
			expectedNeeds: true,
			expectedFields: []diff.FieldChange{
				{Field: "old A", Old: "9.9.9.9 (TTL 300)", New: diff.Absent},
			},
			expectedDeleted: []Record{
				{SubName: "old", Type: "A", Records: []string{"9.9.9.9"}, TTL: 300},
			},
		},
		"ignored subname kept": {
			ignoreSubNames: []string{"ddns"},
			remote: []Record{
				{SubName: "ddns", Type: "A", Records: []string{"9.9.9.9"}, TTL: 60},
			},
		},
		"same subname different type deleted": {
			records: []Record{
				{SubName: "www", Type: "A", Records: []string{"1.2.3.4"}, TTL: 300},
			},
			remote: []Record{
				{SubName: "www", Type: "A", Records: []string{"1.2.3.4"}, TTL: 300},
				{SubName: "www", Type: "TXT", Records: []string{"v=spf1"}, TTL: 300},
			},
			expectedNeeds: true,
			expectedFields: []diff.FieldChange{
				{Field: "www TXT", Old: "v=spf1 (TTL 300)", New: diff.Absent},
			},
			expectedDeleted: []Record{
				{SubName: "www", Type: "TXT", Records: []string{"v=spf1"}, TTL: 300},
			},
		},
		"no records prunes everything": {
			remote: []Record{
				{SubName: "www", Type: "A", Records: []string{"1.2.3.4"}, TTL: 300},
				{SubName: "mail", Type: "MX", Records: []string{"10 mail"}, TTL: 300},
			},
			expectedNeeds: true,
			expectedFields: []diff.FieldChange{
				{Field: "www A", Old: "1.2.3.4 (TTL 300)", New: diff.Absent},
				{Field: "mail MX", Old: "10 mail (TTL 300)", New: diff.Absent},
			},
			expectedDeleted: []Record{
				{SubName: "www", Type: "A", Records: []string{"1.2.3.4"}, TTL: 300},
				{SubName: "mail", Type: "MX", Records: []string{"10 mail"}, TTL: 300},
			},
		},
		"mixed create update delete": {
			records: []Record{
				{SubName: "new", Type: "A", Records: []string{"1.1.1.1"}, TTL: 300},
				{SubName: "www", Type: "A", Records: []string{"1.2.3.4"}, TTL: 300},
			},
			remote: []Record{
				{SubName: "www", Type: "A", Records: []string{"5.6.7.8"}, TTL: 300},
				{SubName: "old", Type: "A", Records: []string{"9.9.9.9"}, TTL: 300},
			},
			expectedNeeds: true,
			expectedFields: []diff.FieldChange{
				{Field: "new A", Old: diff.Absent, New: "1.1.1.1 (TTL 300)"},
				{Field: "www A", Old: "5.6.7.8 (TTL 300)", New: diff.Absent},
				{Field: "www A", Old: diff.Absent, New: "1.2.3.4 (TTL 300)"},
				{Field: "old A", Old: "9.9.9.9 (TTL 300)", New: diff.Absent},
			},
			expectedCreated: []Record{
				{SubName: "new", Type: "A", Records: []string{"1.1.1.1"}, TTL: 300},
			},
			expectedUpdated: []Record{
				{SubName: "www", Type: "A", Records: []string{"1.2.3.4"}, TTL: 300},
			},
			expectedDeleted: []Record{
				{SubName: "old", Type: "A", Records: []string{"9.9.9.9"}, TTL: 300},
			},
		},
	}
}

// fieldChanges asserts d contains exactly the expected field changes.
func fieldChanges(t *testing.T, d tensile.Diff, expected []diff.FieldChange) {
	t.Helper()
	if len(expected) == 0 {
		assert.Nil(t, d)
		return
	}
	changes, ok := d.(*diff.FieldChanges)
	require.True(t, ok, "diff must be *diff.FieldChanges, got %T", d)
	assert.ElementsMatch(t, expected, changes.Fields)
}

func TestZone_NeedsExecution(t *testing.T) {
	t.Parallel()

	for title, cas := range zoneCases() {
		t.Run(title, func(t *testing.T) {
			t.Parallel()
			service := &mockService{records: cas.remote}
			zone := &Zone{
				Domain:         "example.org",
				Service:        service,
				Records:        cas.records,
				IgnoreSubNames: cas.ignoreSubNames,
			}

			needs, d, err := zone.NeedsExecution(&tensile.DefaultWire{})
			require.NoError(t, err)
			assert.Equal(t, cas.expectedNeeds, needs)
			fieldChanges(t, d, cas.expectedFields)
			assert.False(t, service.mutations(), "NeedsExecution must not modify")
		})
	}
}

func TestZone_Execute(t *testing.T) {
	t.Parallel()

	for title, cas := range zoneCases() {
		t.Run(title, func(t *testing.T) {
			t.Parallel()
			service := &mockService{records: cas.remote}
			zone := &Zone{
				Domain:         "example.org",
				Service:        service,
				Records:        cas.records,
				IgnoreSubNames: cas.ignoreSubNames,
			}

			_, err := zone.Execute(&tensile.DefaultWire{})
			require.NoError(t, err)
			assert.ElementsMatch(t, cas.expectedCreated, service.created)
			assert.ElementsMatch(t, cas.expectedUpdated, service.updated)
			assert.ElementsMatch(t, cas.expectedDeleted, service.deleted)
		})
	}
}

func TestZone_ServiceErrors(t *testing.T) {
	t.Parallel()

	errMock := errors.New("mock error")
	// records/remote trigger one create, one update and one delete.
	records := []Record{
		{SubName: "new", Type: "A", Records: []string{"1.1.1.1"}, TTL: 300},
		{SubName: "www", Type: "A", Records: []string{"1.2.3.4"}, TTL: 300},
	}
	remote := []Record{
		{SubName: "www", Type: "A", Records: []string{"5.6.7.8"}, TTL: 300},
		{SubName: "old", Type: "A", Records: []string{"9.9.9.9"}, TTL: 300},
	}

	cases := map[string]struct {
		service     *mockService
		expectedErr string
	}{
		"list error": {
			service:     &mockService{records: remote, listErr: errMock},
			expectedErr: "listing records",
		},
		"create error": {
			service:     &mockService{records: remote, createErr: errMock},
			expectedErr: "creating records",
		},
		"update error": {
			service:     &mockService{records: remote, updateErr: errMock},
			expectedErr: "updating records",
		},
		"delete error": {
			service:     &mockService{records: remote, deleteErr: errMock},
			expectedErr: "deleting records",
		},
	}

	for title, cas := range cases {
		t.Run(title, func(t *testing.T) {
			t.Parallel()
			zone := &Zone{
				Domain:  "example.org",
				Service: cas.service,
				Records: records,
			}

			_, err := zone.Execute(&tensile.DefaultWire{})
			require.ErrorIs(t, err, errMock)
			require.ErrorContains(t, err, cas.expectedErr)

			if cas.service.listErr != nil {
				assert.False(t, cas.service.mutations(), "list error must prevent mutations")
			}
		})
	}
}
