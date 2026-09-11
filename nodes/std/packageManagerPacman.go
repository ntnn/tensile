package std

import (
	"context"
	"fmt"
	"os/exec"
	"strings"
)

var _ PackageManager = (*Pacman)(nil)

func init() {
	RegisterPackageManager("pacman", &Pacman{})
}

// Pacman manages packages through pacman.
type Pacman struct {
	// run executes pacman with args and returns combined output.
	run func(ctx context.Context, args ...string) ([]byte, error)
	// available reports whether pacman is present.
	available func() bool
}

// Handles implements [PackageManager].
// A package is handled when it is known to the database or installed locally.
func (p *Pacman) Handles(ctx context.Context, name string) (bool, error) {
	if !p.isAvailable() {
		return false, nil
	}

	out, err := p.pacman(ctx, "-Si", "--", name)
	if err == nil {
		return true, nil
	}
	if !notFound(out) {
		return false, fmt.Errorf("pacman -Si %q: %w: %s", name, err, strings.TrimSpace(string(out)))
	}

	out, err = p.pacman(ctx, "-Qi", "--", name)
	if err == nil {
		return true, nil
	}
	if !notFound(out) {
		return false, fmt.Errorf("pacman -Qi %q: %w: %s", name, err, strings.TrimSpace(string(out)))
	}
	return false, nil
}

// Installed implements [PackageManager].
func (p *Pacman) Installed(ctx context.Context, name string) (bool, error) {
	out, err := p.pacman(ctx, "-Q", "--", name)
	if err != nil {
		if notFound(out) {
			return false, nil
		}
		return false, fmt.Errorf("pacman -Q %q: %w: %s", name, err, strings.TrimSpace(string(out)))
	}
	return true, nil
}

// Install implements [PackageManager].
func (p *Pacman) Install(ctx context.Context, name string) error {
	out, err := p.pacman(ctx, "-S", "--noconfirm", "--needed", "--", name)
	if err != nil {
		return fmt.Errorf("pacman -S %q: %w: %s", name, err, strings.TrimSpace(string(out)))
	}
	return nil
}

// Remove implements [PackageManager].
func (p *Pacman) Remove(ctx context.Context, name string) error {
	out, err := p.pacman(ctx, "-R", "--noconfirm", "--", name)
	if err != nil {
		return fmt.Errorf("pacman -R %q: %w: %s", name, err, strings.TrimSpace(string(out)))
	}
	return nil
}

func notFound(out []byte) bool {
	return strings.Contains(string(out), "was not found")
}

func (p *Pacman) pacman(ctx context.Context, args ...string) ([]byte, error) {
	if p.run != nil {
		return p.run(ctx, args...)
	}
	//nolint:gosec // args are fixed flags plus the package name
	return exec.CommandContext(ctx, "pacman", args...).CombinedOutput()
}

func (p *Pacman) isAvailable() bool {
	if p.available != nil {
		return p.available()
	}
	_, err := exec.LookPath("pacman")
	return err == nil
}
