package openwrt

import (
	"bytes"
	"context"
	"fmt"
	"os/exec"
	"strings"

	"github.com/ntnn/tensile"
)

var (
	_ tensile.Identifier = (*UCICommit)(nil)
	_ tensile.Executor   = (*UCICommit)(nil)
	_ tensile.Serializer = (*UCICommit)(nil)
)

// UCICommitIdentity returns the identity of the node committing config.
func UCICommitIdentity(config string) tensile.Identity {
	return tensile.AsIdentity("uciCommit", "config", config)
}

// UCICommit commits staged uci changes.
// Should be used as a [tensile.Handler] to be notified by other UCI
// nodes when changed occur.
type UCICommit struct {
	// Config is the config to commit.
	// Empty commits all configs.
	Config string

	// run executes uci with args and returns combined output.
	run func(ctx context.Context, args ...string) ([]byte, error)
}

// Identity implements [tensile.Identifier].
func (u *UCICommit) Identity() tensile.Identity {
	return UCICommitIdentity(u.Config)
}

// SerializesOn implements [tensile.Serializer].
func (u *UCICommit) SerializesOn() []string {
	// A global commit commits all configs, so if a graph contains both
	// UCICommit and UCICommit[config="myconfig"] they need to both lock
	// a common key - `uci` to prevent both mutating the `myconfig`
	// config at the same time.
	if u.Config == "" {
		return []string{"uci"}
	}
	return []string{"uci", uciSerializeKey(u.Config)}
}

// NeedsExecution implements [tensile.Executor].
// Nothing staged means nothing to commit.
func (u *UCICommit) NeedsExecution(c tensile.Wire) (bool, tensile.Diff, error) {
	args := []string{"changes"}
	if u.Config != "" {
		args = append(args, u.Config)
	}
	out, err := u.uci(c.Context(), args...)
	if err != nil {
		return false, nil, fmt.Errorf("uci changes: %w: %s", err, strings.TrimSpace(string(out)))
	}
	return len(bytes.TrimSpace(out)) > 0, nil, nil
}

// Execute implements [tensile.Executor].
func (u *UCICommit) Execute(c tensile.Wire) (tensile.Diff, error) {
	args := []string{"commit"}
	if u.Config != "" {
		args = append(args, u.Config)
	}
	out, err := u.uci(c.Context(), args...)
	if err != nil {
		return nil, fmt.Errorf("uci commit: %w: %s", err, strings.TrimSpace(string(out)))
	}
	return nil, nil //nolint:nilnil // nil Diff is valid
}

func (u *UCICommit) uci(ctx context.Context, args ...string) ([]byte, error) {
	if u.run != nil {
		return u.run(ctx, args...)
	}
	//nolint:gosec // args are fixed verbs plus the managed config
	return exec.CommandContext(ctx, "uci", args...).CombinedOutput()
}
