package std

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"strconv"

	"github.com/ntnn/tensile"
	"github.com/ntnn/tensile/pkg/diff"
	"gopkg.in/ini.v1"
)

var _ tensile.Identifier = (*INIField[string])(nil)
var _ tensile.Validator = (*INIField[string])(nil)
var _ tensile.Depender = (*INIField[string])(nil)
var _ tensile.Serializer = (*INIField[string])(nil)
var _ tensile.Executor = (*INIField[string])(nil)
var _ tensile.Reporter = (*INIField[string])(nil)

// INIValue are the value types storable in an INI key.
// INI stores only strings.
// int is rendered as '<int>'.
// float64 as '<float64>'.
// bool as 'true' or 'false'.
type INIValue interface {
	string | int | float64 | bool
}

// INIFieldOutput is the output reported by [INIField].
type INIFieldOutput struct {
	Path    string
	Section string
	Key     string
	Present bool
	// Value is the current value, empty when absent.
	Value string
}

// INIFieldIdentity returns the identity of the node managing key in section of the INI file at path.
func INIFieldIdentity(path, section, key string) tensile.Identity {
	return tensile.AsIdentity("iniField", "path", path, "section", section, "key", key)
}

// INIField ensures a key in an INI file is present or absent.
// A missing file is created on [tensile.Present].
type INIField[T INIValue] struct {
	// State is the desired state.
	// Defaults to [tensile.Present].
	State tensile.State

	Path string
	// Section is the INI section, empty for the default section.
	Section string
	Key     string
	Value   T

	// LoadOptions configures parsing, empty uses the ini defaults.
	LoadOptions ini.LoadOptions
}

// desired returns the desired state, defaulting to [tensile.Present].
func (i *INIField[T]) desired() tensile.State {
	return i.State.OrDefault()
}

// value renders the declared value to the ini string.
func (i *INIField[T]) value() string {
	switch v := any(i.Value).(type) {
	case string:
		return v
	case int:
		return strconv.Itoa(v)
	case float64:
		return strconv.FormatFloat(v, 'g', -1, 64)
	case bool:
		return strconv.FormatBool(v)
	default:
		// unreachable, INIValue bounds the types
		return ""
	}
}

// field renders Section and Key for diffs.
func (i *INIField[T]) field() string {
	if i.Section == "" {
		return i.Key
	}
	return i.Section + "." + i.Key
}

// Identity implements [tensile.Identifier].
func (i *INIField[T]) Identity() tensile.Identity {
	return INIFieldIdentity(i.Path, i.Section, i.Key)
}

// Validate implements [tensile.Validator].
func (i *INIField[T]) Validate(_ tensile.Wire) error {
	if i.Path == "" {
		return errors.New("path is required")
	}
	if i.Key == "" {
		return errors.New("key is required")
	}
	return i.State.Valid()
}

// DependsOn implements [tensile.Depender].
func (i *INIField[T]) DependsOn() ([]tensile.Identity, error) {
	return append(
		ParentDirIdentities(i.Path),
		FileIdentity(i.Path),
	), nil
}

// SerializesOn implements [tensile.Serializer].
func (i *INIField[T]) SerializesOn() []string {
	return []string{FileSerializeKey(i.Path)}
}

// NeedsExecution implements [tensile.Executor].
func (i *INIField[T]) NeedsExecution(_ tensile.Wire) (bool, tensile.Diff, error) {
	if i.desired() == tensile.ReadOnly {
		return false, nil, nil
	}

	file, err := loadINIFile(i.Path, i.LoadOptions)
	if err != nil {
		return false, nil, err
	}
	current, present := getINIValue(file, i.Section, i.Key)

	if i.desired() == tensile.Absent {
		if !present {
			return false, nil, nil
		}
		return true, diff.NewFieldChanges(&diff.FieldChange{
			Field: i.field(),
			Old:   current,
			New:   diff.Absent,
		}), nil
	}

	if present && current == i.value() {
		return false, nil, nil
	}

	old := diff.Absent
	if present {
		old = current
	}
	return true, diff.NewFieldChanges(&diff.FieldChange{
		Field: i.field(),
		Old:   old,
		New:   i.value(),
	}), nil
}

// Execute implements [tensile.Executor].
func (i *INIField[T]) Execute(_ tensile.Wire) (tensile.Diff, error) {
	file, err := loadINIFile(i.Path, i.LoadOptions)
	if err != nil {
		return nil, err
	}

	switch i.desired() {
	case tensile.Absent:
		file.Section(i.Section).DeleteKey(i.Key)
	case tensile.Present:
		file.Section(i.Section).Key(i.Key).SetValue(i.value())
	case tensile.ReadOnly:
		// unreachable, NeedsExecution reports false
		return nil, nil //nolint:nilnil // nil Diff is valid
	}

	var buf bytes.Buffer
	if _, err := file.WriteTo(&buf); err != nil {
		return nil, fmt.Errorf("marshaling document: %w", err)
	}

	// perms apply only on create, existing files keep theirs
	if err := os.WriteFile(i.Path, buf.Bytes(), 0o644); err != nil { //nolint:gosec,mnd
		return nil, fmt.Errorf("writing file: %w", err)
	}
	return nil, nil //nolint:nilnil // nil Diff is valid
}

// Report implements [tensile.Reporter].
func (i *INIField[T]) Report(_ tensile.Wire) (any, error) {
	file, err := loadINIFile(i.Path, i.LoadOptions)
	if err != nil {
		return nil, err
	}
	value, present := getINIValue(file, i.Section, i.Key)
	return INIFieldOutput{
		Path:    i.Path,
		Section: i.Section,
		Key:     i.Key,
		Present: present,
		Value:   value,
	}, nil
}

// getINIValue returns the key's value and whether it exists.
func getINIValue(file *ini.File, sectionName, key string) (string, bool) {
	section, err := file.GetSection(sectionName)
	if err != nil {
		return "", false
	}
	if !section.HasKey(key) {
		return "", false
	}
	return section.Key(key).String(), true
}

// loadINIFile parses the file, a missing file is an empty document.
func loadINIFile(path string, opts ini.LoadOptions) (*ini.File, error) {
	content, err := os.ReadFile(path) //nolint:gosec // path is the managed file
	if err != nil && !os.IsNotExist(err) {
		return nil, fmt.Errorf("reading file: %w", err)
	}
	file, err := ini.LoadSources(opts, content)
	if err != nil {
		return nil, fmt.Errorf("parsing file: %w", err)
	}
	return file, nil
}
