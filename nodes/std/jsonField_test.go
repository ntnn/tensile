package std

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestJSONField_Validate(t *testing.T) {
	t.Parallel()

	cases := map[string]struct {
		wantErr bool
		node    JSONField
	}{
		"valid":         {false, JSONField{Path: "/f", Key: "a", Value: 1}},
		"valid report":  {false, JSONField{Path: "/f", Key: "a", State: JSONFieldReport}},
		"missing path":  {true, JSONField{Key: "a"}},
		"missing key":   {true, JSONField{Path: "/f"}},
		"unknown state": {true, JSONField{Path: "/f", Key: "a", State: "gone"}},
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

func TestJSONField_NeedsExecution(t *testing.T) {
	t.Parallel()

	cases := map[string]struct {
		expected bool
		onDisk   *string
		node     JSONField
	}{
		"missing file": {
			true,
			nil,
			JSONField{Key: "a", Value: 1},
		},
		"value equal": {
			false,
			new(`{"a": 1}`),
			JSONField{Key: "a", Value: 1},
		},
		"value equal string": {
			false,
			new(`{"a": "x"}`),
			JSONField{Key: "a", Value: "x"},
		},
		"value differs": {
			true,
			new(`{"a": 2}`),
			JSONField{Key: "a", Value: 1},
		},
		"key absent": {
			true,
			new(`{"b": 2}`),
			JSONField{Key: "a", Value: 1},
		},
		"nested equal": {
			false,
			new(`{"a": {"b": 1}}`),
			JSONField{Key: "a.b", Value: 1},
		},
		"nested differs": {
			true,
			new(`{"a": {"b": 2}}`),
			JSONField{Key: "a.b", Value: 1},
		},
		"absent with key present": {
			true,
			new(`{"a": 1}`),
			JSONField{Key: "a", State: JSONFieldAbsent},
		},
		"absent with key absent": {
			false,
			new(`{"b": 1}`),
			JSONField{Key: "a", State: JSONFieldAbsent},
		},
		"absent with missing file": {
			false,
			nil,
			JSONField{Key: "a", State: JSONFieldAbsent},
		},
		"report never executes": {
			false,
			new(`{"a": 2}`),
			JSONField{Key: "a", State: JSONFieldReport, Value: 1},
		},
	}

	for title, cas := range cases {
		t.Run(title, func(t *testing.T) {
			t.Parallel()

			cas.node.Path = filepath.Join(t.TempDir(), "file.json")
			if cas.onDisk != nil {
				require.NoError(t, os.WriteFile(cas.node.Path, []byte(*cas.onDisk), 0o600))
			}

			needs, _, err := cas.node.NeedsExecution(nil)
			require.NoError(t, err)
			assert.Equal(t, cas.expected, needs)
		})
	}
}

func TestJSONField_NeedsExecutionErrors(t *testing.T) {
	t.Parallel()

	cases := map[string]struct {
		onDisk string
		node   JSONField
	}{
		"top level not an object": {`[1]`, JSONField{Key: "a", Value: 1}},
		"invalid json":            {`{`, JSONField{Key: "a", Value: 1}},
		"unmarshalable value":     {`{}`, JSONField{Key: "a", Value: make(chan int)}},
	}

	for title, cas := range cases {
		t.Run(title, func(t *testing.T) {
			t.Parallel()

			cas.node.Path = filepath.Join(t.TempDir(), "file.json")
			require.NoError(t, os.WriteFile(cas.node.Path, []byte(cas.onDisk), 0o600))

			_, _, err := cas.node.NeedsExecution(nil)
			require.Error(t, err)
		})
	}
}

func TestJSONField_Execute(t *testing.T) {
	t.Parallel()

	cases := map[string]struct {
		expected string
		onDisk   *string
		node     JSONField
	}{
		"set value": {
			"{\n  \"a\": 1\n}\n",
			new(`{"a": 2}`),
			JSONField{Key: "a", Value: 1},
		},
		"create file": {
			"{\n  \"a\": 1\n}\n",
			nil,
			JSONField{Key: "a", Value: 1},
		},
		"set nested creates objects": {
			"{\n  \"a\": {\n    \"b\": 1\n  }\n}\n",
			new(`{}`),
			JSONField{Key: "a.b", Value: 1},
		},
		"keeps other fields": {
			"{\n  \"a\": 1,\n  \"b\": 2\n}\n",
			new(`{"b": 2}`),
			JSONField{Key: "a", Value: 1},
		},
		"remove key": {
			"{\n  \"b\": 2\n}\n",
			new(`{"a": 1, "b": 2}`),
			JSONField{Key: "a", State: JSONFieldAbsent},
		},
		"remove nested key": {
			"{\n  \"a\": {}\n}\n",
			new(`{"a": {"b": 1}}`),
			JSONField{Key: "a.b", State: JSONFieldAbsent},
		},
	}

	for title, cas := range cases {
		t.Run(title, func(t *testing.T) {
			t.Parallel()

			cas.node.Path = filepath.Join(t.TempDir(), "file.json")
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

func TestJSONField_ExecuteNonObjectSegment(t *testing.T) {
	t.Parallel()

	path := filepath.Join(t.TempDir(), "file.json")
	require.NoError(t, os.WriteFile(path, []byte(`{"a": 1}`), 0o600))

	node := JSONField{Path: path, Key: "a.b", Value: 1}
	_, err := node.Execute(nil)
	require.Error(t, err)
}

func TestJSONField_Report(t *testing.T) {
	t.Parallel()

	cases := map[string]struct {
		expected JSONFieldOutput
		onDisk   *string
		node     JSONField
	}{
		"present": {
			JSONFieldOutput{Key: "a", Present: true, Value: float64(1)},
			new(`{"a": 1}`),
			JSONField{Key: "a", State: JSONFieldReport},
		},
		"nested present": {
			JSONFieldOutput{Key: "a.b", Present: true, Value: "x"},
			new(`{"a": {"b": "x"}}`),
			JSONField{Key: "a.b", State: JSONFieldReport},
		},
		"absent": {
			JSONFieldOutput{Key: "a", Present: false},
			new(`{"b": 1}`),
			JSONField{Key: "a", State: JSONFieldReport},
		},
		"missing file": {
			JSONFieldOutput{Key: "a", Present: false},
			nil,
			JSONField{Key: "a", State: JSONFieldReport},
		},
	}

	for title, cas := range cases {
		t.Run(title, func(t *testing.T) {
			t.Parallel()

			cas.node.Path = filepath.Join(t.TempDir(), "file.json")
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

func TestSetMapValue(t *testing.T) {
	t.Parallel()

	cases := map[string]struct {
		wantErr  bool
		expected map[string]any
		doc      map[string]any
		key      string
		value    any
	}{
		"top level": {
			false,
			map[string]any{"a": 1},
			map[string]any{},
			"a",
			1,
		},
		"overwrite": {
			false,
			map[string]any{"a": 2},
			map[string]any{"a": 1},
			"a",
			2,
		},
		"keeps siblings": {
			false,
			map[string]any{"a": 1, "b": 2},
			map[string]any{"b": 2},
			"a",
			1,
		},
		"nested existing object": {
			false,
			map[string]any{"a": map[string]any{"b": 1}},
			map[string]any{"a": map[string]any{}},
			"a.b",
			1,
		},
		"nested creates objects": {
			false,
			map[string]any{"a": map[string]any{"b": map[string]any{"c": 1}}},
			map[string]any{},
			"a.b.c",
			1,
		},
		"overwrite object": {
			false,
			map[string]any{"a": 1},
			map[string]any{"a": map[string]any{"b": 2}},
			"a",
			1,
		},
		"non-object segment": {
			true,
			nil,
			map[string]any{"a": 1},
			"a.b",
			2,
		},
	}

	for title, cas := range cases {
		t.Run(title, func(t *testing.T) {
			t.Parallel()

			err := setMapValue(cas.doc, cas.key, cas.value)
			if cas.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, cas.expected, cas.doc)
		})
	}
}

func TestDeleteMapValue(t *testing.T) {
	t.Parallel()

	cases := map[string]struct {
		expected map[string]any
		doc      map[string]any
		key      string
	}{
		"top level": {
			map[string]any{},
			map[string]any{"a": 1},
			"a",
		},
		"keeps siblings": {
			map[string]any{"b": 2},
			map[string]any{"a": 1, "b": 2},
			"a",
		},
		"nested": {
			map[string]any{"a": map[string]any{}},
			map[string]any{"a": map[string]any{"b": 1}},
			"a.b",
		},
		"missing key is a no-op": {
			map[string]any{"b": 2},
			map[string]any{"b": 2},
			"a",
		},
		"non-object segment is a no-op": {
			map[string]any{"a": 1},
			map[string]any{"a": 1},
			"a.b",
		},
		"missing segment is a no-op": {
			map[string]any{},
			map[string]any{},
			"a.b",
		},
	}

	for title, cas := range cases {
		t.Run(title, func(t *testing.T) {
			t.Parallel()

			deleteMapValue(cas.doc, cas.key)
			assert.Equal(t, cas.expected, cas.doc)
		})
	}
}

func TestNormalizeJSONValue(t *testing.T) {
	t.Parallel()

	cases := map[string]struct {
		wantErr  bool
		expected any
		value    any
	}{
		"int to float64":    {false, float64(1), 1},
		"float64 unchanged": {false, float64(1.5), 1.5},
		"string unchanged":  {false, "x", "x"},
		"bool unchanged":    {false, true, true},
		"nil unchanged":     {false, nil, nil},
		"int slice":         {false, []any{float64(1), float64(2)}, []int{1, 2}},
		"typed map":         {false, map[string]any{"a": float64(1)}, map[string]int{"a": 1}},
		"unmarshalable":     {true, nil, make(chan int)},
	}

	for title, cas := range cases {
		t.Run(title, func(t *testing.T) {
			t.Parallel()

			normalized, err := normalizeJSONValue(cas.value)
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
// renderJSONValue keeps the JSON quoting.
func TestJSONField_NeedsExecutionDiff(t *testing.T) {
	t.Parallel()

	path := filepath.Join(t.TempDir(), "file.json")
	require.NoError(t, os.WriteFile(path, []byte(`{"a": "1"}`), 0o600))

	node := JSONField{Path: path, Key: "a", Value: 1}
	needs, d, err := node.NeedsExecution(nil)
	require.NoError(t, err)
	require.True(t, needs, "string \"1\" and number 1 differ")
	require.NotNil(t, d)
	assert.Equal(t, `a: "1" -> 1`, d.String())
}

func TestRenderJSONValue(t *testing.T) {
	t.Parallel()

	cases := map[string]struct {
		expected string
		value    any
	}{
		"int":        {"1", 1},
		"string one": {`"1"`, "1"},
		"float":      {"1.5", 1.5},
		"string":     {`"x"`, "x"},
		"bool":       {"true", true},
		"nil":        {"null", nil},
		"slice":      {"[1,2]", []int{1, 2}},
		"object":     {`{"a":1}`, map[string]any{"a": 1}},
	}

	for title, cas := range cases {
		t.Run(title, func(t *testing.T) {
			t.Parallel()

			assert.Equal(t, cas.expected, renderJSONValue(cas.value))
		})
	}
}
