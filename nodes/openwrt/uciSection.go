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
	_ tensile.Identifier = (*UCISection)(nil)
	_ tensile.Validator  = (*UCISection)(nil)
	_ tensile.Depender   = (*UCISection)(nil)
	_ tensile.Notifier   = (*UCISection)(nil)
	_ tensile.Executor   = (*UCISection)(nil)
)

// UCISectionIdentity returns the identity of the node managing the named section.
func UCISectionIdentity(config, section string) tensile.Identity {
	return tensile.AsIdentity("uciSection", "config", config, "section", section)
}

// UCISection ensures a named UCI section exists with the given type.
// Changes are only staged, not committed.
type UCISection struct {
	Config  string
	Section string
	// Type is the section type, required unless [UCISection.State] is [UCIAbsent].
	Type string
	// State is the desired presence.
	// Empty means [UCIPresent].
	State UCIState

	// run executes uci with args and returns combined output.
	run func(ctx context.Context, args ...string) ([]byte, error)
}

// Validate implements [tensile.Validator].
func (s *UCISection) Validate(_ tensile.Wire) error {
	if s.Config == "" {
		return errors.New("config is required")
	}
	if s.Section == "" {
		return errors.New("section is required")
	}
	switch s.desired() {
	case UCIPresent:
		if s.Type == "" {
			return errors.New("type is required when state is present")
		}
	case UCIAbsent:
	default:
		return fmt.Errorf("unknown state %q", s.State)
	}
	return nil
}

// Identity implements [tensile.Identifier].
func (s *UCISection) Identity() tensile.Identity {
	return UCISectionIdentity(s.Config, s.Section)
}

// DependsOn implements [tensile.Depender].
func (s *UCISection) DependsOn() ([]tensile.Identity, error) {
	return []tensile.Identity{
		UCIDeleteAnonymousSectionsIdentity(s.Config, s.Type),
	}, nil
}

// Notifies implements [tensile.Notifier].
// Staged changes notify the config's commit and the global one.
func (s *UCISection) Notifies() ([]tensile.Identity, error) {
	return []tensile.Identity{
		UCICommitIdentity(s.Config),
		UCICommitIdentity(""),
	}, nil
}

// NeedsExecution implements [tensile.Executor].
func (s *UCISection) NeedsExecution(c tensile.Wire) (bool, error) {
	sectionType, exists, err := s.current(c.Context())
	if err != nil {
		return false, err
	}

	if s.desired() == UCIAbsent {
		return exists, nil
	}
	return !exists || sectionType != s.Type, nil
}

// Execute implements [tensile.Executor].
func (s *UCISection) Execute(c tensile.Wire) error {
	ctx := c.Context()

	if s.desired() == UCIAbsent {
		out, err := s.uci(ctx, "delete", s.path())
		if err != nil && !uciNotFound(out) {
			return fmt.Errorf("uci delete %q: %w: %s", s.path(), err, strings.TrimSpace(string(out)))
		}
		return nil
	}

	out, err := s.uci(ctx, "set", s.path()+"="+s.Type)
	if err != nil {
		return fmt.Errorf("uci set %q: %w: %s", s.path(), err, strings.TrimSpace(string(out)))
	}
	return nil
}

// current returns the section's type and whether the section exists.
func (s *UCISection) current(ctx context.Context) (string, bool, error) {
	out, err := s.uci(ctx, "get", s.path())
	if err != nil {
		if uciNotFound(out) {
			return "", false, nil
		}
		return "", false, fmt.Errorf("uci get %q: %w: %s", s.path(), err, strings.TrimSpace(string(out)))
	}
	return strings.TrimSpace(string(out)), true, nil
}

func (s *UCISection) path() string {
	return s.Config + "." + s.Section
}

func (s *UCISection) desired() UCIState {
	if s.State == "" {
		return UCIPresent
	}
	return s.State
}

func (s *UCISection) uci(ctx context.Context, args ...string) ([]byte, error) {
	if s.run != nil {
		return s.run(ctx, args...)
	}
	//nolint:gosec // args are fixed verbs plus managed uci paths
	return exec.CommandContext(ctx, "uci", args...).CombinedOutput()
}
