package std

import (
	"bytes"
	"encoding/json/jsontext"
	json "encoding/json/v2"
	"errors"
	"fmt"
	"os"
	"reflect"
	"strings"

	"github.com/ntnn/tensile"
	"github.com/ntnn/tensile/pkg/diff"
)

var _ tensile.Identifier = (*JSONField)(nil)
var _ tensile.Validator = (*JSONField)(nil)
var _ tensile.Depender = (*JSONField)(nil)
var _ tensile.Executor = (*JSONField)(nil)
var _ tensile.Reporter = (*JSONField)(nil)

// JSONFieldOutput is the output reported by [JSONField].
type JSONFieldOutput struct {
	Path    string
	Key     string
	Present bool
	// Value is the current value, nil when absent.
	Value any
}

// JSONField ensures a field in a JSON file is present or absent.
// The top level of the file must be a JSON object.
// Key is a dot-separated path into nested objects.
// A missing file is created on [tensile.Present].
type JSONField struct {
	// State is the desired state.
	// Defaults to [tensile.Present].
	State tensile.State

	Path  string
	Key   string
	Value any
}

// desired returns the desired state, defaulting to [tensile.Present].
func (j *JSONField) desired() tensile.State {
	return j.State.OrDefault()
}

// Identity implements [tensile.Identifier].
func (j *JSONField) Identity() tensile.Identity {
	return tensile.AsIdentity("jsonField", "path", j.Path, "key", j.Key)
}

// Validate implements [tensile.Validator].
func (j *JSONField) Validate(_ tensile.Wire) error {
	if j.Path == "" {
		return errors.New("path is required")
	}
	if j.Key == "" {
		return errors.New("key is required")
	}
	return j.State.Valid()
}

// DependsOn implements [tensile.Depender].
func (j *JSONField) DependsOn() ([]tensile.Identity, error) {
	return append(
		ParentDirIdentities(j.Path),
		FileIdentity(j.Path),
	), nil
}

// NeedsExecution implements [tensile.Executor].
func (j *JSONField) NeedsExecution(_ tensile.Wire) (bool, tensile.Diff, error) {
	if j.desired() == tensile.ReadOnly {
		return false, nil, nil
	}

	doc, err := loadJSONFile(j.Path)
	if err != nil {
		return false, nil, err
	}
	current, present := getMapValue(doc, j.Key)

	if j.desired() == tensile.Absent {
		if !present {
			return false, nil, nil
		}
		return true, diff.NewFieldChanges(&diff.FieldChange{
			Field: j.Key,
			Old:   renderJSONValue(current),
			New:   diff.Absent,
		}), nil
	}

	desired, err := normalizeJSONValue(j.Value)
	if err != nil {
		return false, nil, err
	}
	if present && reflect.DeepEqual(current, desired) {
		return false, nil, nil
	}

	old := diff.Absent
	if present {
		old = renderJSONValue(current)
	}
	return true, diff.NewFieldChanges(&diff.FieldChange{
		Field: j.Key,
		Old:   old,
		New:   renderJSONValue(desired),
	}), nil
}

// Execute implements [tensile.Executor].
func (j *JSONField) Execute(_ tensile.Wire) (tensile.Diff, error) {
	doc, err := loadJSONFile(j.Path)
	if err != nil {
		return nil, err
	}

	switch j.desired() {
	case tensile.Absent:
		deleteMapValue(doc, j.Key)
	case tensile.Present:
		if err := setMapValue(doc, j.Key, j.Value); err != nil {
			return nil, err
		}
	case tensile.ReadOnly:
		// unreachable, NeedsExecution reports false
		return nil, nil //nolint:nilnil // nil Diff is valid
	}

	// Deterministic sorts object keys for stable output across runs
	out, err := json.Marshal(doc, json.Deterministic(true), jsontext.WithIndent("  "))
	if err != nil {
		return nil, fmt.Errorf("marshaling document: %w", err)
	}
	out = append(out, '\n')

	// perms apply only on create, existing files keep theirs
	if err := os.WriteFile(j.Path, out, 0o644); err != nil { //nolint:gosec,mnd
		return nil, fmt.Errorf("writing file: %w", err)
	}
	return nil, nil //nolint:nilnil // nil Diff is valid
}

// Report implements [tensile.Reporter].
func (j *JSONField) Report(_ tensile.Wire) (any, error) {
	doc, err := loadJSONFile(j.Path)
	if err != nil {
		return nil, err
	}
	value, present := getMapValue(doc, j.Key)
	return JSONFieldOutput{
		Path:    j.Path,
		Key:     j.Key,
		Present: present,
		Value:   value,
	}, nil
}

// loadJSONFile parses the file, a missing or empty file is an empty map.
func loadJSONFile(path string) (map[string]any, error) {
	content, err := os.ReadFile(path) //nolint:gosec // path is the managed file
	if err != nil && !os.IsNotExist(err) {
		return nil, fmt.Errorf("reading file: %w", err)
	}
	doc := map[string]any{}
	if len(bytes.TrimSpace(content)) == 0 {
		return doc, nil
	}
	if err := json.Unmarshal(content, &doc); err != nil {
		return nil, fmt.Errorf("parsing file: %w", err)
	}
	return doc, nil
}

// getMapValue returns the value at the dot-separated key and whether it exists.
func getMapValue(doc map[string]any, key string) (any, bool) {
	var current any = doc
	for segment := range strings.SplitSeq(key, ".") {
		object, ok := current.(map[string]any)
		if !ok {
			return nil, false
		}
		current, ok = object[segment]
		if !ok {
			return nil, false
		}
	}
	return current, true
}

// setMapValue sets value at the dot-separated key.
// Missing intermediate maps are created.
// Errors when a segment holds a non-map.
func setMapValue(doc map[string]any, key string, value any) error {
	segments := strings.Split(key, ".")
	object := doc
	for _, segment := range segments[:len(segments)-1] {
		next, ok := object[segment]
		if !ok {
			child := map[string]any{}
			object[segment] = child
			object = child
			continue
		}
		child, ok := next.(map[string]any)
		if !ok {
			return fmt.Errorf("key %q: segment %q holds a non-map value", key, segment)
		}
		object = child
	}
	object[segments[len(segments)-1]] = value
	return nil
}

// deleteMapValue removes the dot-separated key, a missing key or non-map segment is a no-op.
func deleteMapValue(doc map[string]any, key string) {
	segments := strings.Split(key, ".")
	object := doc
	for _, segment := range segments[:len(segments)-1] {
		child, ok := object[segment].(map[string]any)
		if !ok {
			return
		}
		object = child
	}
	delete(object, segments[len(segments)-1])
}

// normalizeJSONValue roundtrips value through the json encoder/decoder so types match
// unmarshaled documents, e.g. int(8080) equals float64(8080).
func normalizeJSONValue(value any) (any, error) {
	raw, err := json.Marshal(value)
	if err != nil {
		return nil, fmt.Errorf("marshaling value: %w", err)
	}
	var normalized any
	if err := json.Unmarshal(raw, &normalized); err != nil {
		return nil, fmt.Errorf("unmarshaling value: %w", err)
	}
	return normalized, nil
}

// renderJSONValue returns the value as compact JSON for diffs.
func renderJSONValue(value any) string {
	raw, err := json.Marshal(value)
	if err != nil {
		return fmt.Sprintf("%v", value)
	}
	return string(raw)
}
