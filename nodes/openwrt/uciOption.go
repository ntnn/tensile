package openwrt

import (
	"context"
	"errors"
	"fmt"
	"os/exec"
	"slices"
	"strconv"
	"strings"

	"github.com/ntnn/tensile"
	"github.com/ntnn/tensile/pkg/diff"
)

var (
	_ tensile.Identifier = (*UCIOption[string])(nil)
	_ tensile.Validator  = (*UCIOption[string])(nil)
	_ tensile.Depender   = (*UCIOption[string])(nil)
	_ tensile.Notifier   = (*UCIOption[string])(nil)
	_ tensile.Executor   = (*UCIOption[string])(nil)
	_ tensile.Serializer = (*UCIOption[string])(nil)
)

// UCIValue are the value types storable in a UCI option.
// UCI stores only strings or lists of strings.
// int is represented as '<int>', bool as '0' or '1' (or 'off'/'on', ...).
type UCIValue interface {
	string | int | bool | []string | []int
}

// UCIOptionIdentity returns the identity of the node managing the named option.
func UCIOptionIdentity(config, section, option string) tensile.Identity {
	return tensile.AsIdentity("uciOption", "config", config, "section", section, "option", option)
}

// UCIOption ensures a UCI option has the given value.
// Changes are only staged, not committed.
// The section must exist.
type UCIOption[T UCIValue] struct {
	Config  string
	Section string
	Option  string
	Value   T

	// State is the desired presence.
	// Empty means [UCIPresent].
	State UCIState

	// run executes uci with args and returns combined output.
	run func(ctx context.Context, args ...string) ([]byte, error)
}

// Validate implements [tensile.Validator].
func (o *UCIOption[T]) Validate(_ tensile.Wire) error {
	if o.Config == "" {
		return errors.New("config is required")
	}
	if o.Section == "" {
		return errors.New("section is required")
	}
	if o.Option == "" {
		return errors.New("option is required")
	}
	switch o.desired() {
	case UCIPresent, UCIAbsent:
	default:
		return fmt.Errorf("unknown state %q", o.State)
	}
	return nil
}

// Identity implements [tensile.Identifier].
func (o *UCIOption[T]) Identity() tensile.Identity {
	return UCIOptionIdentity(o.Config, o.Section, o.Option)
}

// DependsOn implements [tensile.Depender].
func (o *UCIOption[T]) DependsOn() ([]tensile.Identity, error) {
	return []tensile.Identity{
		UCISectionIdentity(o.Config, o.Section),
	}, nil
}

// Notifies implements [tensile.Notifier].
// Staged changes notify the config's commit and the global one.
func (o *UCIOption[T]) Notifies() ([]tensile.Identity, error) {
	return []tensile.Identity{
		UCICommitIdentity(o.Config),
		UCICommitIdentity(""),
	}, nil
}

// SerializesOn implements [tensile.Serializer].
func (o *UCIOption[T]) SerializesOn() []string {
	return []string{uciSerializeKey(o.Config)}
}

// NeedsExecution implements [tensile.Executor].
func (o *UCIOption[T]) NeedsExecution(c tensile.Wire) (bool, tensile.Diff, error) {
	current, exists, err := o.current(c.Context())
	if err != nil {
		return false, nil, err
	}

	d := &diff.FieldChange{
		Field: o.path(),
		Old:   diff.Absent,
		New:   diff.Absent,
	}

	if exists {
		d.Old = strings.Join(current, ", ")
	}

	if o.desired() == UCIAbsent {
		if !exists {
			return false, nil, nil
		}
		return true, diff.NewFieldChanges(d), nil
	}

	if exists && slices.Equal(current, o.values()) {
		return false, nil, nil
	}
	d.New = strings.Join(o.values(), ", ")

	return true, diff.NewFieldChanges(d), nil
}

// Execute implements [tensile.Executor].
func (o *UCIOption[T]) Execute(c tensile.Wire) (tensile.Diff, error) {
	ctx := c.Context()

	if o.desired() == UCIAbsent {
		return nil, o.delete(ctx)
	}

	if !o.isList() {
		out, err := o.uci(ctx, "set", o.path()+"="+o.values()[0])
		if err != nil {
			return nil, fmt.Errorf("uci set %q: %w: %s", o.path(), err, strings.TrimSpace(string(out)))
		}
		return nil, nil //nolint:nilnil // nil Diff is valid
	}

	if err := o.delete(ctx); err != nil {
		return nil, err
	}
	for _, value := range o.values() {
		out, err := o.uci(ctx, "add_list", o.path()+"="+value)
		if err != nil {
			return nil, fmt.Errorf("uci add_list %q: %w: %s", o.path(), err, strings.TrimSpace(string(out)))
		}
	}
	return nil, nil //nolint:nilnil // nil Diff is valid
}

// current returns the option's values and whether the option exists.
func (o *UCIOption[T]) current(ctx context.Context) ([]string, bool, error) {
	out, err := o.uci(ctx, "show", o.path())
	if err != nil {
		if uciNotFound(out) {
			return nil, false, nil
		}
		return nil, false, fmt.Errorf("uci show %q: %w: %s", o.path(), err, strings.TrimSpace(string(out)))
	}

	line, _, _ := strings.Cut(string(out), "\n")
	_, rendered, ok := strings.Cut(line, "=")
	if !ok {
		return nil, false, fmt.Errorf("uci show %q: unexpected output %q", o.path(), line)
	}
	return parseUCIValues(rendered), true, nil
}

// delete stages the option's deletion, a missing option is a no-op.
func (o *UCIOption[T]) delete(ctx context.Context) error {
	out, err := o.uci(ctx, "delete", o.path())
	if err != nil && !uciNotFound(out) {
		return fmt.Errorf("uci delete %q: %w: %s", o.path(), err, strings.TrimSpace(string(out)))
	}
	return nil
}

// values renders the declared value to uci strings.
func (o *UCIOption[T]) values() []string {
	switch v := any(o.Value).(type) {
	case string:
		return []string{v}
	case int:
		return []string{strconv.Itoa(v)}
	case bool:
		if v {
			return []string{"1"}
		}
		return []string{"0"}
	case []string:
		return v
	case []int:
		values := make([]string, len(v))
		for i, n := range v {
			values[i] = strconv.Itoa(n)
		}
		return values
	default:
		// unreachable, UCIValue bounds the types
		return nil
	}
}

func (o *UCIOption[T]) isList() bool {
	switch any(o.Value).(type) {
	case []string, []int:
		return true
	default:
		return false
	}
}

func (o *UCIOption[T]) path() string {
	return o.Config + "." + o.Section + "." + o.Option
}

func (o *UCIOption[T]) desired() UCIState {
	if o.State == "" {
		return UCIPresent
	}
	return o.State
}

func (o *UCIOption[T]) uci(ctx context.Context, args ...string) ([]byte, error) {
	if o.run != nil {
		return o.run(ctx, args...)
	}
	//nolint:gosec // args are fixed verbs plus managed uci paths
	return exec.CommandContext(ctx, "uci", args...).CombinedOutput()
}
