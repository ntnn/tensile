package systemd

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/coreos/go-systemd/v22/unit"
	"github.com/ntnn/tensile"
	"github.com/ntnn/tensile/nodes/std"
	"github.com/ntnn/tensile/pkg/queue"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// testSections renders to testContent.
var testSections = []*unit.UnitSection{
	{
		Section: "Unit",
		Entries: []*unit.UnitEntry{
			{Name: "Description", Value: "foo"},
		},
	},
	{
		Section: "Service",
		Entries: []*unit.UnitEntry{
			{Name: "ExecStart", Value: "/bin/foo"},
		},
	},
}

const testContent = "[Unit]\nDescription=foo\n\n[Service]\nExecStart=/bin/foo\n"

func TestUnit_Validate(t *testing.T) {
	t.Parallel()

	cases := map[string]struct {
		wantErr bool
		node    Unit
	}{
		"valid":                {false, Unit{Name: "foo.service", Sections: testSections}},
		"valid absent":         {false, Unit{Name: "foo.service", State: tensile.Absent}},
		"valid drop-in":        {false, Unit{Name: "foo.service", DropIn: "10-x", Sections: testSections}},
		"missing name":         {true, Unit{}},
		"missing name drop-in": {true, Unit{DropIn: "10-x"}},
		"unknown state":        {true, Unit{Name: "foo.service", State: "gone"}},
	}

	for title, cas := range cases {
		t.Run(title, func(t *testing.T) {
			t.Parallel()

			err := cas.node.Validate(nil)
			if cas.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
		})
	}
}

func TestUnit_Path(t *testing.T) {
	t.Parallel()

	assert.Equal(t,
		filepath.FromSlash("/etc/systemd/system/foo.service"),
		(&Unit{Name: "foo.service"}).Path(),
	)
	assert.Equal(t,
		filepath.FromSlash("/x/foo.service"),
		(&Unit{Name: "foo.service", Dir: "/x"}).Path(),
	)
	assert.Equal(t,
		filepath.FromSlash("/etc/systemd/system/foo.service.d/10-x.conf"),
		(&Unit{Name: "foo.service", DropIn: "10-x"}).Path(),
	)
}

func TestUnit_Execute(t *testing.T) {
	t.Parallel()

	cases := map[string]struct {
		expected *string
		onDisk   *string
		state    tensile.State
		dropIn   string
	}{
		"create":                 {new(testContent), nil, tensile.Present, ""},
		"overwrite":              {new(testContent), new("[Unit]\nDescription=bar\n"), tensile.Present, ""},
		"remove":                 {nil, new(testContent), tensile.Absent, ""},
		"remove missing":         {nil, nil, tensile.Absent, ""},
		"create drop-in":         {new(testContent), nil, tensile.Present, "10-x"},
		"overwrite drop-in":      {new(testContent), new("[Unit]\nDescription=bar\n"), tensile.Present, "10-x"},
		"remove drop-in":         {nil, new(testContent), tensile.Absent, "10-x"},
		"remove missing drop-in": {nil, nil, tensile.Absent, "10-x"},
	}

	for title, cas := range cases {
		t.Run(title, func(t *testing.T) {
			t.Parallel()

			node := Unit{
				State:    cas.state,
				Name:     "foo.service",
				DropIn:   cas.dropIn,
				Dir:      filepath.Join(t.TempDir(), "system"),
				Sections: testSections,
			}
			if cas.onDisk != nil {
				require.NoError(t, os.MkdirAll(filepath.Dir(node.Path()), 0o750))
				require.NoError(t, os.WriteFile(node.Path(), []byte(*cas.onDisk), 0o600))
			}

			_, err := node.Execute(nil)
			require.NoError(t, err)

			content, err := os.ReadFile(node.Path())
			if cas.expected == nil {
				require.ErrorIs(t, err, os.ErrNotExist)
			} else {
				require.NoError(t, err)
				assert.Equal(t, *cas.expected, string(content))
			}

			needs, _, err := node.NeedsExecution(nil)
			require.NoError(t, err)
			assert.False(t, needs, "execute must be idempotent")
		})
	}
}

func TestUnit_NeedsExecution(t *testing.T) {
	t.Parallel()

	cases := map[string]struct {
		expected     bool
		expectedDiff string
		onDisk       *string
		state        tensile.State
	}{
		"missing file": {
			true,
			"--- PATH\n+++ PATH\n@@ -0,0 +1,5 @@\n+[Unit]\n+Description=foo\n+\n+[Service]\n+ExecStart=/bin/foo\n",
			nil,
			tensile.Present,
		},
		"equal": {false, "", new(testContent), tensile.Present},
		"differs": {
			true,
			"--- PATH\n+++ PATH\n@@ -1,2 +1,5 @@\n [Unit]\n-Description=bar\n+Description=foo\n" +
				"+\n+[Service]\n+ExecStart=/bin/foo\n",
			new("[Unit]\nDescription=bar\n"),
			tensile.Present,
		},
		"absent present": {true, "file: present -> (absent)", new(testContent), tensile.Absent},
		"absent missing": {false, "", nil, tensile.Absent},
		"read-only":      {false, "", new("x"), tensile.ReadOnly},
	}

	for title, cas := range cases {
		t.Run(title, func(t *testing.T) {
			t.Parallel()

			node := Unit{
				State:    cas.state,
				Name:     "foo.service",
				Dir:      t.TempDir(),
				Sections: testSections,
			}
			if cas.onDisk != nil {
				require.NoError(t, os.WriteFile(node.Path(), []byte(*cas.onDisk), 0o600))
			}

			needs, d, err := node.NeedsExecution(nil)
			require.NoError(t, err)
			assert.Equal(t, cas.expected, needs)
			if cas.expectedDiff == "" {
				return
			}
			require.NotNil(t, d)
			assert.Equal(t, strings.ReplaceAll(cas.expectedDiff, "PATH", node.Path()), d.String())
		})
	}
}

func TestUnit_conflictsWithFile(t *testing.T) {
	t.Parallel()

	node := &Unit{Name: "foo.service"}
	q := queue.New()
	q.Add(node, &std.File{Path: node.Path()})
	_, err := q.Build()
	require.ErrorContains(t, err, "already claimed")
}
