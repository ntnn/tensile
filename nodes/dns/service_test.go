package dns

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestMergeRecords(t *testing.T) {
	t.Parallel()

	cases := map[string]struct {
		records  []Record
		expected []Record
	}{
		"distinct subname and type kept": {
			records: []Record{
				{SubName: "www", Type: "A", Records: []string{"1.2.3.4"}},
				{SubName: "www", Type: "TXT", Records: []string{"v=spf1"}},
			},
			expected: []Record{
				{SubName: "www", Type: "A", Records: []string{"1.2.3.4"}},
				{SubName: "www", Type: "TXT", Records: []string{"v=spf1"}},
			},
		},
		"same subname and type merged": {
			records: []Record{
				{SubName: "www", Type: "A", Records: []string{"1.2.3.4"}},
				{SubName: "www", Type: "A", Records: []string{"5.6.7.8"}},
			},
			expected: []Record{
				{SubName: "www", Type: "A", Records: []string{"1.2.3.4", "5.6.7.8"}},
			},
		},
		"duplicate entries deduplicated": {
			records: []Record{
				{SubName: "www", Type: "A", Records: []string{"1.2.3.4"}},
				{SubName: "www", Type: "A", Records: []string{"1.2.3.4", "5.6.7.8"}},
			},
			expected: []Record{
				{SubName: "www", Type: "A", Records: []string{"1.2.3.4", "5.6.7.8"}},
			},
		},
		"lowest managed ttl wins": {
			records: []Record{
				{SubName: "www", Type: "A", Records: []string{"1.2.3.4"}},
				{SubName: "www", Type: "A", Records: []string{"5.6.7.8"}, TTL: 300},
				{SubName: "www", Type: "A", Records: []string{"9.9.9.9"}, TTL: 600},
			},
			expected: []Record{
				{SubName: "www", Type: "A", Records: []string{"1.2.3.4", "5.6.7.8", "9.9.9.9"}, TTL: 300},
			},
		},
		"lowest managed ttl wins lowest last": {
			records: []Record{
				{SubName: "www", Type: "A", Records: []string{"1.2.3.4"}},
				{SubName: "www", Type: "A", Records: []string{"9.9.9.9"}, TTL: 600},
				{SubName: "www", Type: "A", Records: []string{"8.8.8.8"}, TTL: 400},
				{SubName: "www", Type: "A", Records: []string{"5.6.7.8"}, TTL: 300},
			},
			expected: []Record{
				{SubName: "www", Type: "A", Records: []string{"1.2.3.4", "5.6.7.8", "8.8.8.8", "9.9.9.9"}, TTL: 300},
			},
		},
		"lowest managed ttl wins lowest in the middle": {
			records: []Record{
				{SubName: "www", Type: "A", Records: []string{"1.2.3.4"}},
				{SubName: "www", Type: "A", Records: []string{"9.9.9.9"}, TTL: 600},
				{SubName: "www", Type: "A", Records: []string{"5.6.7.8"}, TTL: 300},
				{SubName: "www", Type: "A", Records: []string{"8.8.8.8"}, TTL: 400},
			},
			expected: []Record{
				{SubName: "www", Type: "A", Records: []string{"1.2.3.4", "5.6.7.8", "8.8.8.8", "9.9.9.9"}, TTL: 300},
			},
		},
		"case variants merged": {
			records: []Record{
				{SubName: "www", Type: "A", Records: []string{"1.2.3.4"}},
				{SubName: "WWW", Type: "a", Records: []string{"5.6.7.8"}},
			},
			expected: []Record{
				{SubName: "www", Type: "A", Records: []string{"1.2.3.4", "5.6.7.8"}},
			},
		},
		"unmerged entries sorted and deduplicated": {
			records: []Record{
				{SubName: "www", Type: "A", Records: []string{"5.6.7.8", "1.2.3.4", "1.2.3.4"}},
			},
			expected: []Record{
				{SubName: "www", Type: "A", Records: []string{"1.2.3.4", "5.6.7.8"}},
			},
		},
		"result sorted by subname and type": {
			records: []Record{
				{SubName: "www", Type: "TXT", Records: []string{"v=spf1"}},
				{SubName: "www", Type: "A", Records: []string{"1.2.3.4"}},
				{SubName: "mail", Type: "MX", Records: []string{"10 mail"}},
			},
			expected: []Record{
				{SubName: "mail", Type: "MX", Records: []string{"10 mail"}},
				{SubName: "www", Type: "A", Records: []string{"1.2.3.4"}},
				{SubName: "www", Type: "TXT", Records: []string{"v=spf1"}},
			},
		},
	}

	for title, cas := range cases {
		t.Run(title, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, cas.expected, MergeRecords(cas.records))
		})
	}
}

func TestSplitRecords(t *testing.T) {
	t.Parallel()

	cases := map[string]struct {
		records  []Record
		expected []Record
	}{
		"single entry unchanged": {
			records: []Record{
				{SubName: "www", Type: "A", Records: []string{"1.2.3.4"}, TTL: 300},
			},
			expected: []Record{
				{SubName: "www", Type: "A", Records: []string{"1.2.3.4"}, TTL: 300},
			},
		},
		"multiple entries split": {
			records: []Record{
				{SubName: "www", Type: "A", Records: []string{"1.2.3.4", "5.6.7.8"}, TTL: 300},
			},
			expected: []Record{
				{SubName: "www", Type: "A", Records: []string{"1.2.3.4"}, TTL: 300},
				{SubName: "www", Type: "A", Records: []string{"5.6.7.8"}, TTL: 300},
			},
		},
		"multiple records split": {
			records: []Record{
				{SubName: "www", Type: "A", Records: []string{"1.2.3.4", "5.6.7.8"}},
				{SubName: "www", Type: "TXT", Records: []string{"v=spf1"}},
			},
			expected: []Record{
				{SubName: "www", Type: "A", Records: []string{"1.2.3.4"}},
				{SubName: "www", Type: "A", Records: []string{"5.6.7.8"}},
				{SubName: "www", Type: "TXT", Records: []string{"v=spf1"}},
			},
		},
		"no entries yields nothing": {
			records: []Record{
				{SubName: "www", Type: "A"},
			},
			expected: []Record{},
		},
	}

	for title, cas := range cases {
		t.Run(title, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, cas.expected, SplitRecords(cas.records))
		})
	}
}
