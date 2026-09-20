package openwrt

import (
	"context"
	"errors"
	"fmt"
	"os/exec"
	"strings"

	"github.com/ntnn/tensile"
)

var (
	_ tensile.Identifier = (*UCIDeleteAnonymousSections)(nil)
	_ tensile.Validator  = (*UCIDeleteAnonymousSections)(nil)
	_ tensile.Notifier   = (*UCIDeleteAnonymousSections)(nil)
	_ tensile.Executor   = (*UCIDeleteAnonymousSections)(nil)
)

// UCIDeleteAnonymousSectionsIdentity returns the identity of the UCIDeleteAnonymousSectionsIdentity node.
func UCIDeleteAnonymousSectionsIdentity(config, sectionType string) tensile.Identity {
	return tensile.AsIdentity("ucideleteanonymoussections", "config", config, "type", sectionType)
}

// UCIDeleteAnonymousSections deletes all anonymous sections of a type.
// Deletions are only staged, not committed.
// Useful to delete e.g. defaults coming from a package.
type UCIDeleteAnonymousSections struct {
	Config string
	// Type is the section type to delete anonymous sections of.
	Type string

	// run executes uci with args and returns combined output.
	run func(ctx context.Context, args ...string) ([]byte, error)
}

// Validate implements [tensile.Validator].
func (d *UCIDeleteAnonymousSections) Validate(_ tensile.Wire) error {
	if d.Config == "" {
		return errors.New("config is required")
	}
	if d.Type == "" {
		return errors.New("type is required")
	}
	return nil
}

// Identity implements [tensile.Identifier].
func (d *UCIDeleteAnonymousSections) Identity() tensile.Identity {
	return UCIDeleteAnonymousSectionsIdentity(d.Config, d.Type)
}

// Notifies implements [tensile.Notifier].
// Staged deletions notify the config's commit and the global one.
func (d *UCIDeleteAnonymousSections) Notifies() ([]tensile.Identity, error) {
	return []tensile.Identity{
		UCICommitIdentity(d.Config),
		UCICommitIdentity(""),
	}, nil
}

// NeedsExecution implements [tensile.Executor].
func (d *UCIDeleteAnonymousSections) NeedsExecution(c tensile.Wire) (bool, error) {
	paths, err := d.anonymous(c.Context())
	if err != nil {
		return false, err
	}
	return len(paths) > 0, nil
}

// Execute implements [tensile.Executor].
func (d *UCIDeleteAnonymousSections) Execute(c tensile.Wire) error {
	ctx := c.Context()

	paths, err := d.anonymous(ctx)
	if err != nil {
		return err
	}

	// gather resolved IDs of anonymous sections
	ids := make([]string, 0, len(paths))
	for _, path := range paths {
		id, err := d.resolve(ctx, path)
		if err != nil {
			return err
		}
		ids = append(ids, id)
	}

	// delete by the resolved IDs
	for _, id := range ids {
		out, err := d.uci(ctx, "delete", d.Config+"."+id)
		if err != nil {
			return fmt.Errorf("uci delete %q: %w: %s",
				d.Config+"."+id, err, strings.TrimSpace(string(out)))
		}
	}
	return nil
}

// anonymous returns the extended paths (config.@type[i]) of all anonymous sections of the type.
// A missing config means no sections.
func (d *UCIDeleteAnonymousSections) anonymous(ctx context.Context) ([]string, error) {
	out, err := d.uci(ctx, "show", d.Config)
	if err != nil {
		if uciNotFound(out) {
			return nil, nil
		}
		return nil, fmt.Errorf("uci show %q: %w: %s",
			d.Config, err, strings.TrimSpace(string(out)))
	}

	// section lines are config.@type[i]=type, option lines have a
	// .option suffix before the =
	prefix := d.Config + ".@" + d.Type + "["
	var paths []string
	for line := range strings.Lines(string(out)) {
		path, _, ok := strings.Cut(strings.TrimSpace(line), "=")
		if !ok {
			continue
		}
		if strings.HasPrefix(path, prefix) && strings.HasSuffix(path, "]") {
			paths = append(paths, path)
		}
	}
	return paths, nil
}

// resolve returns the generated section id behind an extended path.
// uci show on a section prints config.<id>=type as the first line.
func (d *UCIDeleteAnonymousSections) resolve(ctx context.Context, path string) (string, error) {
	out, err := d.uci(ctx, "show", path)
	if err != nil {
		return "", fmt.Errorf("uci show %q: %w: %s",
			path, err, strings.TrimSpace(string(out)))
	}

	line, _, _ := strings.Cut(string(out), "\n")
	ref, _, ok := strings.Cut(line, "=")
	if !ok {
		return "", fmt.Errorf("uci show %q: unexpected output %q", path, line)
	}
	id := strings.TrimPrefix(ref, d.Config+".")
	if id == ref || id == "" {
		return "", fmt.Errorf("uci show %q: unexpected section reference %q", path, ref)
	}
	return id, nil
}

func (d *UCIDeleteAnonymousSections) uci(ctx context.Context, args ...string) ([]byte, error) {
	if d.run != nil {
		return d.run(ctx, args...)
	}
	//nolint:gosec // args are fixed verbs plus managed uci paths
	return exec.CommandContext(ctx, "uci", args...).CombinedOutput()
}
