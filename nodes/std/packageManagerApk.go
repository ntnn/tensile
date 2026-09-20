package std

import (
	"context"
	"errors"
	"fmt"
	"os/exec"
	"strings"
)

var _ PackageManager = (*Apk)(nil)

func init() {
	RegisterPackageManager("apk", &Apk{})
}

// Apk manages packages through apk.
type Apk struct {
	// run executes apk with args and returns combined output.
	run func(ctx context.Context, args ...string) ([]byte, error)
	// available reports whether apk is present.
	available func() bool
}

// Handles implements [PackageManager].
func (a *Apk) Handles(ctx context.Context, name string) (bool, error) {
	if !a.isAvailable() {
		return false, nil
	}

	out, err := a.apk(ctx, "info", name)
	if err == nil {
		return true, nil
	}
	if exitedNonZero(err) {
		return false, nil
	}
	return false, fmt.Errorf("apk info %q: %w: %s", name, err, strings.TrimSpace(string(out)))
}

// Installed implements [PackageManager].
func (a *Apk) Installed(ctx context.Context, name string) (bool, error) {
	out, err := a.apk(ctx, "info", "-e", name)
	if err == nil {
		return true, nil
	}
	if exitedNonZero(err) {
		return false, nil
	}
	return false, fmt.Errorf("apk info -e %q: %w: %s", name, err, strings.TrimSpace(string(out)))
}

// Install implements [PackageManager].
func (a *Apk) Install(ctx context.Context, name string) error {
	out, err := a.apk(ctx, "add", name)
	if err != nil {
		return fmt.Errorf("apk add %q: %w: %s", name, err, strings.TrimSpace(string(out)))
	}
	return nil
}

// Remove implements [PackageManager].
func (a *Apk) Remove(ctx context.Context, name string) error {
	out, err := a.apk(ctx, "del", name)
	if err != nil {
		return fmt.Errorf("apk del %q: %w: %s", name, err, strings.TrimSpace(string(out)))
	}
	return nil
}

// exitedNonZero reports whether err is a command exit with non-zero code,
// as opposed to a failure to run the command at all.
func exitedNonZero(err error) bool {
	var exitErr *exec.ExitError
	return errors.As(err, &exitErr)
}

func (a *Apk) apk(ctx context.Context, args ...string) ([]byte, error) {
	if a.run != nil {
		return a.run(ctx, args...)
	}
	//nolint:gosec // args are fixed verbs plus the package name
	return exec.CommandContext(ctx, "apk", args...).CombinedOutput()
}

func (a *Apk) isAvailable() bool {
	if a.available != nil {
		return a.available()
	}
	_, err := exec.LookPath("apk")
	return err == nil
}
