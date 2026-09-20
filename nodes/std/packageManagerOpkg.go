package std

import (
	"bytes"
	"context"
	"fmt"
	"os/exec"
	"strings"
)

var _ PackageManager = (*Opkg)(nil)

func init() {
	RegisterPackageManager("opkg", &Opkg{})
}

// Opkg manages packages through opkg.
type Opkg struct {
	// run executes opkg with args and returns combined output.
	run func(ctx context.Context, args ...string) ([]byte, error)
	// available reports whether opkg is present.
	available func() bool
}

// Handles implements [PackageManager].
// A package is handled when it is known to the package lists or installed locally.
func (o *Opkg) Handles(ctx context.Context, name string) (bool, error) {
	if !o.isAvailable() {
		return false, nil
	}

	out, err := o.opkg(ctx, "info", name)
	if err != nil {
		return false, fmt.Errorf("opkg info %q: %w: %s", name, err, strings.TrimSpace(string(out)))
	}
	if len(bytes.TrimSpace(out)) > 0 {
		return true, nil
	}

	out, err = o.opkg(ctx, "status", name)
	if err != nil {
		return false, fmt.Errorf("opkg status %q: %w: %s", name, err, strings.TrimSpace(string(out)))
	}
	return len(bytes.TrimSpace(out)) > 0, nil
}

// Installed implements [PackageManager].
func (o *Opkg) Installed(ctx context.Context, name string) (bool, error) {
	out, err := o.opkg(ctx, "status", name)
	if err != nil {
		return false, fmt.Errorf("opkg status %q: %w: %s", name, err, strings.TrimSpace(string(out)))
	}
	return strings.Contains(string(out), "Status: install"), nil
}

// Install implements [PackageManager].
func (o *Opkg) Install(ctx context.Context, name string) error {
	out, err := o.opkg(ctx, "install", name)
	if err != nil {
		return fmt.Errorf("opkg install %q: %w: %s", name, err, strings.TrimSpace(string(out)))
	}
	return nil
}

// Remove implements [PackageManager].
func (o *Opkg) Remove(ctx context.Context, name string) error {
	out, err := o.opkg(ctx, "remove", name)
	if err != nil {
		return fmt.Errorf("opkg remove %q: %w: %s", name, err, strings.TrimSpace(string(out)))
	}
	return nil
}

func (o *Opkg) opkg(ctx context.Context, args ...string) ([]byte, error) {
	if o.run != nil {
		return o.run(ctx, args...)
	}
	//nolint:gosec // args are fixed verbs plus the package name
	return exec.CommandContext(ctx, "opkg", args...).CombinedOutput()
}

func (o *Opkg) isAvailable() bool {
	if o.available != nil {
		return o.available()
	}
	_, err := exec.LookPath("opkg")
	return err == nil
}
