package std

import (
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/ntnn/tensile"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestYAMLField_Validate(t *testing.T) {
	t.Parallel()

	cases := map[string]struct {
		wantErr bool
		node    YAMLField
	}{
		"valid":         {false, YAMLField{Path: "/f", Key: "a", Value: 1}},
		"valid report":  {false, YAMLField{Path: "/f", Key: "a", State: tensile.ReadOnly}},
		"missing path":  {true, YAMLField{Key: "a"}},
		"missing key":   {true, YAMLField{Path: "/f"}},
		"unknown state": {true, YAMLField{Path: "/f", Key: "a", State: "gone"}},
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

func TestYAMLField_NeedsExecution(t *testing.T) {
	t.Parallel()

	cases := map[string]struct {
		expected bool
		onDisk   *string
		node     YAMLField
	}{
		"missing file": {
			true,
			nil,
			YAMLField{Key: "a", Value: 1},
		},
		"value equal": {
			false,
			new("a: 1\n"),
			YAMLField{Key: "a", Value: 1},
		},
		"value equal string": {
			false,
			new("a: x\n"),
			YAMLField{Key: "a", Value: "x"},
		},
		"value differs": {
			true,
			new("a: 2\n"),
			YAMLField{Key: "a", Value: 1},
		},
		"key absent": {
			true,
			new("b: 2\n"),
			YAMLField{Key: "a", Value: 1},
		},
		"nested equal": {
			false,
			new("a:\n  b: 1\n"),
			YAMLField{Key: "a.b", Value: 1},
		},
		"nested differs": {
			true,
			new("a:\n  b: 2\n"),
			YAMLField{Key: "a.b", Value: 1},
		},
		"absent with key present": {
			true,
			new("a: 1\n"),
			YAMLField{Key: "a", State: tensile.Absent},
		},
		"absent with key absent": {
			false,
			new("b: 1\n"),
			YAMLField{Key: "a", State: tensile.Absent},
		},
		"absent with missing file": {
			false,
			nil,
			YAMLField{Key: "a", State: tensile.Absent},
		},
		"report never executes": {
			false,
			new("a: 2\n"),
			YAMLField{Key: "a", State: tensile.ReadOnly, Value: 1},
		},
	}

	for title, cas := range cases {
		t.Run(title, func(t *testing.T) {
			t.Parallel()

			cas.node.Path = filepath.Join(t.TempDir(), "file.yaml")
			if cas.onDisk != nil {
				require.NoError(t, os.WriteFile(cas.node.Path, []byte(*cas.onDisk), 0o600))
			}

			needs, _, err := cas.node.NeedsExecution(nil)
			require.NoError(t, err)
			assert.Equal(t, cas.expected, needs)
		})
	}
}

// yamlMarshalFailer errors on marshal, yaml.Marshal panics on truly
// unmarshalable types like channels instead of returning an error.
type yamlMarshalFailer struct{}

func (yamlMarshalFailer) MarshalYAML() (any, error) {
	return nil, errors.New("fail")
}

func TestYAMLField_NeedsExecutionErrors(t *testing.T) {
	t.Parallel()

	cases := map[string]struct {
		onDisk string
		node   YAMLField
	}{
		"top level not a mapping": {"- 1\n", YAMLField{Key: "a", Value: 1}},
		"invalid yaml":            {"{", YAMLField{Key: "a", Value: 1}},
		"unmarshalable value":     {"{}\n", YAMLField{Key: "a", Value: yamlMarshalFailer{}}},
	}

	for title, cas := range cases {
		t.Run(title, func(t *testing.T) {
			t.Parallel()

			cas.node.Path = filepath.Join(t.TempDir(), "file.yaml")
			require.NoError(t, os.WriteFile(cas.node.Path, []byte(cas.onDisk), 0o600))

			_, _, err := cas.node.NeedsExecution(nil)
			require.Error(t, err)
		})
	}
}

func TestYAMLField_Execute(t *testing.T) {
	t.Parallel()

	cases := map[string]struct {
		expected string
		onDisk   *string
		node     YAMLField
	}{
		"set value": {
			"a: 1\n",
			new("a: 2\n"),
			YAMLField{Key: "a", Value: 1},
		},
		"create file": {
			"a: 1\n",
			nil,
			YAMLField{Key: "a", Value: 1},
		},
		"set nested creates mappings": {
			"a:\n    b: 1\n",
			new("{}\n"),
			YAMLField{Key: "a.b", Value: 1},
		},
		"keeps other fields": {
			"a: 1\nb: 2\n",
			new("b: 2\n"),
			YAMLField{Key: "a", Value: 1},
		},
		"remove key": {
			"b: 2\n",
			new("a: 1\nb: 2\n"),
			YAMLField{Key: "a", State: tensile.Absent},
		},
		"remove nested key": {
			"a: {}\n",
			new("a:\n  b: 1\n"),
			YAMLField{Key: "a.b", State: tensile.Absent},
		},
	}

	for title, cas := range cases {
		t.Run(title, func(t *testing.T) {
			t.Parallel()

			cas.node.Path = filepath.Join(t.TempDir(), "file.yaml")
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

func TestYAMLField_ExecuteNonMappingSegment(t *testing.T) {
	t.Parallel()

	path := filepath.Join(t.TempDir(), "file.yaml")
	require.NoError(t, os.WriteFile(path, []byte("a: 1\n"), 0o600))

	node := YAMLField{Path: path, Key: "a.b", Value: 1}
	_, err := node.Execute(nil)
	require.Error(t, err)
}

func TestYAMLField_Report(t *testing.T) {
	t.Parallel()

	cases := map[string]struct {
		expected YAMLFieldOutput
		onDisk   *string
		node     YAMLField
	}{
		"present": {
			YAMLFieldOutput{Key: "a", Present: true, Value: 1},
			new("a: 1\n"),
			YAMLField{Key: "a", State: tensile.ReadOnly},
		},
		"nested present": {
			YAMLFieldOutput{Key: "a.b", Present: true, Value: "x"},
			new("a:\n  b: x\n"),
			YAMLField{Key: "a.b", State: tensile.ReadOnly},
		},
		"absent": {
			YAMLFieldOutput{Key: "a", Present: false},
			new("b: 1\n"),
			YAMLField{Key: "a", State: tensile.ReadOnly},
		},
		"missing file": {
			YAMLFieldOutput{Key: "a", Present: false},
			nil,
			YAMLField{Key: "a", State: tensile.ReadOnly},
		},
	}

	for title, cas := range cases {
		t.Run(title, func(t *testing.T) {
			t.Parallel()

			cas.node.Path = filepath.Join(t.TempDir(), "file.yaml")
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

func TestNormalizeYAMLValue(t *testing.T) {
	t.Parallel()

	cases := map[string]struct {
		wantErr  bool
		expected any
		value    any
	}{
		"int unchanged":     {false, 1, 1},
		"uint to int":       {false, 1, uint(1)},
		"float64 unchanged": {false, 1.5, 1.5},
		"string unchanged":  {false, "x", "x"},
		"bool unchanged":    {false, true, true},
		"nil unchanged":     {false, nil, nil},
		"int slice":         {false, []any{1, 2}, []int{1, 2}},
		"typed map":         {false, map[string]any{"a": 1}, map[string]int{"a": 1}},
		"unmarshalable":     {true, nil, yamlMarshalFailer{}},
	}

	for title, cas := range cases {
		t.Run(title, func(t *testing.T) {
			t.Parallel()

			normalized, err := normalizeYAMLValue(cas.value)
			if cas.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, cas.expected, normalized)
		})
	}
}

// NeedsExecution diffs must distinguish string "1" from number 1,
// renderYAMLValue keeps the YAML quoting.
func TestYAMLField_NeedsExecutionDiff(t *testing.T) {
	t.Parallel()

	path := filepath.Join(t.TempDir(), "file.yaml")
	require.NoError(t, os.WriteFile(path, []byte("a: \"1\"\n"), 0o600))

	node := YAMLField{Path: path, Key: "a", Value: 1}
	needs, d, err := node.NeedsExecution(nil)
	require.NoError(t, err)
	require.True(t, needs, "string \"1\" and number 1 differ")
	require.NotNil(t, d)
	assert.Equal(t, `a: "1" -> 1`, d.String())
}

func TestRenderYAMLValue(t *testing.T) {
	t.Parallel()

	cases := map[string]struct {
		expected string
		value    any
	}{
		"int":        {"1", 1},
		"string one": {`"1"`, "1"},
		"float":      {"1.5", 1.5},
		"string":     {"x", "x"},
		"bool":       {"true", true},
		"nil":        {"null", nil},
		"slice":      {"- 1\n- 2", []int{1, 2}},
		"mapping":    {"a: 1", map[string]any{"a": 1}},
	}

	for title, cas := range cases {
		t.Run(title, func(t *testing.T) {
			t.Parallel()

			assert.Equal(t, cas.expected, renderYAMLValue(cas.value))
		})
	}
}
