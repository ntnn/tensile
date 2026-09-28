package std

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"reflect"
	"strings"

	"github.com/ntnn/tensile"
	"github.com/ntnn/tensile/pkg/diff"
	"go.yaml.in/yaml/v3"
)

var _ tensile.Identifier = (*YAMLField)(nil)
var _ tensile.Validator = (*YAMLField)(nil)
var _ tensile.Depender = (*YAMLField)(nil)
var _ tensile.Executor = (*YAMLField)(nil)
var _ tensile.Reporter = (*YAMLField)(nil)

// YAMLFieldState is the desired state of a field in a YAML file.
type YAMLFieldState string

// Desired states for [YAMLField].
const (
	YAMLFieldPresent YAMLFieldState = "present"
	YAMLFieldAbsent  YAMLFieldState = "absent"
	// YAMLFieldReport only lets the node report the current value.
	YAMLFieldReport YAMLFieldState = "report"
)

// YAMLFieldOutput is the output reported by [YAMLField].
type YAMLFieldOutput struct {
	Path    string
	Key     string
	Present bool
	// Value is the current value, nil when absent.
	Value any
}

// YAMLField ensures a field in a YAML file is present or absent.
// The top level of the file must be a YAML mapping.
// Key is a dot-separated path into nested mappings.
// A missing file is created on [YAMLFieldPresent].
type YAMLField struct {
	// State is the desired state.
	// Defaults to [YAMLFieldPresent].
	State YAMLFieldState

	Path  string
	Key   string
	Value any
}

// desired returns the desired state, defaulting to [YAMLFieldPresent].
func (y *YAMLField) desired() YAMLFieldState {
	if y.State == "" {
		return YAMLFieldPresent
	}
	return y.State
}

// Identity implements [tensile.Identifier].
func (y *YAMLField) Identity() tensile.Identity {
	return tensile.AsIdentity("yamlField", "path", y.Path, "key", y.Key)
}

// Validate implements [tensile.Validator].
func (y *YAMLField) Validate(_ tensile.Wire) error {
	if y.Path == "" {
		return errors.New("path is required")
	}
	if y.Key == "" {
		return errors.New("key is required")
	}
	switch y.State {
	case "", YAMLFieldPresent, YAMLFieldAbsent, YAMLFieldReport:
	default:
		return fmt.Errorf("unknown state %q", y.State)
	}
	return nil
}

// DependsOn implements [tensile.Depender].
func (y *YAMLField) DependsOn() ([]tensile.Identity, error) {
	return append(
		ParentDirIdentities(y.Path),
		FileIdentity(y.Path),
	), nil
}

// NeedsExecution implements [tensile.Executor].
func (y *YAMLField) NeedsExecution(_ tensile.Wire) (bool, tensile.Diff, error) {
	if y.desired() == YAMLFieldReport {
		return false, nil, nil
	}

	doc, err := loadYAMLFile(y.Path)
	if err != nil {
		return false, nil, err
	}
	current, present := getMapValue(doc, y.Key)

	if y.desired() == YAMLFieldAbsent {
		if !present {
			return false, nil, nil
		}
		return true, diff.NewFieldChanges(&diff.FieldChange{
			Field: y.Key,
			Old:   renderYAMLValue(current),
			New:   diff.Absent,
		}), nil
	}

	desired, err := normalizeYAMLValue(y.Value)
	if err != nil {
		return false, nil, err
	}
	if present && reflect.DeepEqual(current, desired) {
		return false, nil, nil
	}

	old := diff.Absent
	if present {
		old = renderYAMLValue(current)
	}
	return true, diff.NewFieldChanges(&diff.FieldChange{
		Field: y.Key,
		Old:   old,
		New:   renderYAMLValue(desired),
	}), nil
}

// Execute implements [tensile.Executor].
func (y *YAMLField) Execute(_ tensile.Wire) (tensile.Diff, error) {
	doc, err := loadYAMLFile(y.Path)
	if err != nil {
		return nil, err
	}

	switch y.desired() {
	case YAMLFieldAbsent:
		deleteMapValue(doc, y.Key)
	case YAMLFieldPresent:
		if err := setMapValue(doc, y.Key, y.Value); err != nil {
			return nil, err
		}
	case YAMLFieldReport:
		// unreachable, NeedsExecution reports false
		return nil, nil //nolint:nilnil // nil Diff is valid
	}

	out, err := yaml.Marshal(doc)
	if err != nil {
		return nil, fmt.Errorf("marshaling document: %w", err)
	}

	// perms apply only on create, existing files keep theirs
	if err := os.WriteFile(y.Path, out, 0o644); err != nil { //nolint:gosec,mnd
		return nil, fmt.Errorf("writing file: %w", err)
	}
	return nil, nil //nolint:nilnil // nil Diff is valid
}

// Report implements [tensile.Reporter].
func (y *YAMLField) Report(_ tensile.Wire) (any, error) {
	doc, err := loadYAMLFile(y.Path)
	if err != nil {
		return nil, err
	}
	value, present := getMapValue(doc, y.Key)
	return YAMLFieldOutput{
		Path:    y.Path,
		Key:     y.Key,
		Present: present,
		Value:   value,
	}, nil
}

// loadYAMLFile parses the file, a missing or empty file is an empty map.
func loadYAMLFile(path string) (map[string]any, error) {
	content, err := os.ReadFile(path) //nolint:gosec // path is the managed file
	if err != nil && !os.IsNotExist(err) {
		return nil, fmt.Errorf("reading file: %w", err)
	}
	doc := map[string]any{}
	if len(bytes.TrimSpace(content)) == 0 {
		return doc, nil
	}
	if err := yaml.Unmarshal(content, &doc); err != nil {
		return nil, fmt.Errorf("parsing file: %w", err)
	}
	return doc, nil
}

// normalizeYAMLValue roundtrips value through the yaml encoder/decoder so types match
// unmarshaled documents, e.g. uint(8080) equals int(8080).
func normalizeYAMLValue(value any) (any, error) {
	raw, err := yaml.Marshal(value)
	if err != nil {
		return nil, fmt.Errorf("marshaling value: %w", err)
	}
	var normalized any
	if err := yaml.Unmarshal(raw, &normalized); err != nil {
		return nil, fmt.Errorf("unmarshaling value: %w", err)
	}
	return normalized, nil
}

// renderYAMLValue returns the value as YAML for diffs, multiline for non-scalars.
func renderYAMLValue(value any) string {
	raw, err := yaml.Marshal(value)
	if err != nil {
		return fmt.Sprintf("%v", value)
	}
	return strings.TrimSpace(string(raw))
}
