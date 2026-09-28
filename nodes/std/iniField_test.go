package std

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestINIField_Validate(t *testing.T) {
	t.Parallel()

	cases := map[string]struct {
		wantErr bool
		node    INIField[string]
	}{
		"valid":         {false, INIField[string]{Path: "/f", Key: "a", Value: "1"}},
		"valid section": {false, INIField[string]{Path: "/f", Section: "s", Key: "a", Value: "1"}},
		"valid report":  {false, INIField[string]{Path: "/f", Key: "a", State: INIFieldReport}},
		"missing path":  {true, INIField[string]{Key: "a"}},
		"missing key":   {true, INIField[string]{Path: "/f"}},
		"unknown state": {true, INIField[string]{Path: "/f", Key: "a", State: "gone"}},
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

func TestINIField_NeedsExecution(t *testing.T) {
	t.Parallel()

	cases := map[string]struct {
		expected bool
		onDisk   *string
		node     INIField[string]
	}{
		"missing file": {
			true,
			nil,
			INIField[string]{Key: "a", Value: "1"},
		},
		"value equal": {
			false,
			new("a = 1\n"),
			INIField[string]{Key: "a", Value: "1"},
		},
		"value differs": {
			true,
			new("a = 2\n"),
			INIField[string]{Key: "a", Value: "1"},
		},
		"key absent": {
			true,
			new("b = 2\n"),
			INIField[string]{Key: "a", Value: "1"},
		},
		"section value equal": {
			false,
			new("[s]\na = 1\n"),
			INIField[string]{Section: "s", Key: "a", Value: "1"},
		},
		"section absent": {
			true,
			new("a = 1\n"),
			INIField[string]{Section: "s", Key: "a", Value: "1"},
		},
		"key in other section": {
			true,
			new("[o]\na = 1\n"),
			INIField[string]{Section: "s", Key: "a", Value: "1"},
		},
		"absent with key present": {
			true,
			new("a = 1\n"),
			INIField[string]{Key: "a", State: INIFieldAbsent},
		},
		"absent with key absent": {
			false,
			new("b = 1\n"),
			INIField[string]{Key: "a", State: INIFieldAbsent},
		},
		"absent with missing file": {
			false,
			nil,
			INIField[string]{Key: "a", State: INIFieldAbsent},
		},
		"report never executes": {
			false,
			new("a = 2\n"),
			INIField[string]{Key: "a", State: INIFieldReport, Value: "1"},
		},
	}

	for title, cas := range cases {
		t.Run(title, func(t *testing.T) {
			t.Parallel()

			cas.node.Path = filepath.Join(t.TempDir(), "file.ini")
			if cas.onDisk != nil {
				require.NoError(t, os.WriteFile(cas.node.Path, []byte(*cas.onDisk), 0o600))
			}

			needs, _, err := cas.node.NeedsExecution(nil)
			require.NoError(t, err)
			assert.Equal(t, cas.expected, needs)
		})
	}
}

// NeedsExecution must compare the rendered int against the file value.
func TestINIField_NeedsExecutionInt(t *testing.T) {
	t.Parallel()

	path := filepath.Join(t.TempDir(), "file.ini")
	require.NoError(t, os.WriteFile(path, []byte("a = 1\n"), 0o600))

	node := INIField[int]{Path: path, Key: "a", Value: 1}
	needs, _, err := node.NeedsExecution(nil)
	require.NoError(t, err)
	assert.False(t, needs)
}

func TestINIField_Execute(t *testing.T) {
	t.Parallel()

	cases := map[string]struct {
		expected string
		onDisk   *string
		node     INIField[string]
	}{
		"set value": {
			"a = 1\n",
			new("a = 2\n"),
			INIField[string]{Key: "a", Value: "1"},
		},
		"create file": {
			"a = 1\n",
			nil,
			INIField[string]{Key: "a", Value: "1"},
		},
		"set in section": {
			"[s]\na = 1\n",
			nil,
			INIField[string]{Section: "s", Key: "a", Value: "1"},
		},
		"keeps other keys": {
			"b = 2\na = 1\n",
			new("b = 2\n"),
			INIField[string]{Key: "a", Value: "1"},
		},
		"keeps other sections": {
			"[o]\nb = 2\n\n[s]\na = 1\n",
			new("[o]\nb = 2\n"),
			INIField[string]{Section: "s", Key: "a", Value: "1"},
		},
		"remove key": {
			"b = 2\n",
			new("a = 1\nb = 2\n"),
			INIField[string]{Key: "a", State: INIFieldAbsent},
		},
		"remove key from section": {
			"[s]\nb = 2\n",
			new("[s]\na = 1\nb = 2\n"),
			INIField[string]{Section: "s", Key: "a", State: INIFieldAbsent},
		},
	}

	for title, cas := range cases {
		t.Run(title, func(t *testing.T) {
			t.Parallel()

			cas.node.Path = filepath.Join(t.TempDir(), "file.ini")
			if cas.onDisk != nil {
				require.NoError(t, os.WriteFile(cas.node.Path, []byte(*cas.onDisk), 0o600))
			}

			_, err := cas.node.Execute(nil)
			require.NoError(t, err)

			content, err := os.ReadFile(cas.node.Path)
			require.NoError(t, err)
			assert.Equal(t, cas.expected, string(content))

			needs, _, err := cas.node.NeedsExecution(nil)
			require.NoError(t, err)
			assert.False(t, needs, "execute must be idempotent")
		})
	}
}

func TestINIField_ExecuteInt(t *testing.T) {
	t.Parallel()

	path := filepath.Join(t.TempDir(), "file.ini")
	node := INIField[int]{Path: path, Key: "a", Value: 1}
	_, err := node.Execute(nil)
	require.NoError(t, err)

	content, err := os.ReadFile(path) //nolint:gosec // path from t.TempDir
	require.NoError(t, err)
	assert.Equal(t, "a = 1\n", string(content))
}

func TestINIField_Report(t *testing.T) {
	t.Parallel()

	cases := map[string]struct {
		expected INIFieldOutput
		onDisk   *string
		node     INIField[string]
	}{
		"present": {
			INIFieldOutput{Key: "a", Present: true, Value: "1"},
			new("a = 1\n"),
			INIField[string]{Key: "a", State: INIFieldReport},
		},
		"present in section": {
			INIFieldOutput{Section: "s", Key: "a", Present: true, Value: "x"},
			new("[s]\na = x\n"),
			INIField[string]{Section: "s", Key: "a", State: INIFieldReport},
		},
		"absent": {
			INIFieldOutput{Key: "a", Present: false},
			new("b = 1\n"),
			INIField[string]{Key: "a", State: INIFieldReport},
		},
		"missing file": {
			INIFieldOutput{Key: "a", Present: false},
			nil,
			INIField[string]{Key: "a", State: INIFieldReport},
		},
	}

	for title, cas := range cases {
		t.Run(title, func(t *testing.T) {
			t.Parallel()

			cas.node.Path = filepath.Join(t.TempDir(), "file.ini")
			cas.expected.Path = cas.node.Path
			if cas.onDisk != nil {
				require.NoError(t, os.WriteFile(cas.node.Path, []byte(*cas.onDisk), 0o600))
			}

			output, err := cas.node.Report(nil)
			require.NoError(t, err)
			assert.Equal(t, cas.expected, output)
		})
	}
}

// Diffs prefix the key with the section.
func TestINIField_NeedsExecutionDiff(t *testing.T) {
	t.Parallel()

	path := filepath.Join(t.TempDir(), "file.ini")
	require.NoError(t, os.WriteFile(path, []byte("[s]\na = 1\n"), 0o600))

	node := INIField[string]{Path: path, Section: "s", Key: "a", Value: "2"}
	needs, d, err := node.NeedsExecution(nil)
	require.NoError(t, err)
	require.True(t, needs)
	require.NotNil(t, d)
	assert.Equal(t, "s.a: 1 -> 2", d.String())
}

// Table impossible, the generic type parameter differs per case.
func TestINIField_value(t *testing.T) {
	t.Parallel()

	assert.Equal(t, "x", (&INIField[string]{Value: "x"}).value())
	assert.Equal(t, "1", (&INIField[int]{Value: 1}).value())
	assert.Equal(t, "1.5", (&INIField[float64]{Value: 1.5}).value())
	assert.Equal(t, "0.1", (&INIField[float64]{Value: 0.1}).value())
	assert.Equal(t, "true", (&INIField[bool]{Value: true}).value())
	assert.Equal(t, "false", (&INIField[bool]{Value: false}).value())
}
