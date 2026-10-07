package std

import (
	"errors"
	"fmt"

	"github.com/ntnn/tensile"
	"github.com/ntnn/tensile/pkg/diff"
)

var _ tensile.Identifier = (*Package)(nil)
var _ tensile.Validator = (*Package)(nil)
var _ tensile.Depender = (*Package)(nil)
var _ tensile.Serializer = (*Package)(nil)
var _ tensile.Executor = (*Package)(nil)

// PackageIdentity returns the identity of the node managing the named package.
func PackageIdentity(name string) tensile.Identity {
	return tensile.AsIdentity("package", "name", name)
}

// Package ensures a package is present or absent.
//
// Package deliberately does not manage package versions.
// Version constraints should be handled either in the package manger
// configuration or in the repository the packages are sourced from
// (e.g. a Spacewalk, SUSE Manager, ...) should produce a prefiltered
// stream.
// Similarly updates should be installed through a controlled process,
// not in a provisioning tool.
type Package struct {
	Name string
	// State is the desired presence.
	// Empty means [tensile.Present].
	// [tensile.ReadOnly] is not supported.
	State tensile.State
	// Manager selects a registered [PackageManager] by name.
	// Empty means detection.
	Manager string
}

// Validate implements [tensile.Validator].
func (p *Package) Validate(_ tensile.Wire) error {
	if p.Name == "" {
		return errors.New("name is required")
	}
	switch p.State { //nolint:exhaustive // ReadOnly is unsupported, rejected by default
	case "", tensile.Present, tensile.Absent:
	default:
		return fmt.Errorf("unknown state %q", p.State)
	}
	return nil
}

// Identity implements [tensile.Identifier].
func (p *Package) Identity() tensile.Identity {
	return PackageIdentity(p.Name)
}

// DependsOn implements [tensile.Depender].
// Package lists are updated first when a [PackageManagerUpdate] node is enqueued.
func (p *Package) DependsOn() ([]tensile.Identity, error) {
	return []tensile.Identity{PackageManagerUpdateIdentity()}, nil
}

// SerializesOn implements [tensile.Serializer].
// If a manager is set explicitly the serialization key is per manager.
// If it is empty the key is shared for all package manager.
func (p *Package) SerializesOn() []string {
	if p.Manager != "" {
		return []string{"package-" + p.Manager}
	}
	return []string{"package"}
}

// NeedsExecution implements [tensile.Executor].
// If no registered [PackageManager] handles the package the package is
// considered not installed.
func (p *Package) NeedsExecution(c tensile.Wire) (bool, tensile.Diff, error) {
	mgr, err := p.manager(c)
	if _, ok := errors.AsType[*noManagerError](err); ok {
		return p.diff(false)
	}
	if err != nil {
		return false, nil, err
	}

	installed, err := mgr.Installed(c.Context(), p.Name)
	if err != nil {
		return false, nil, fmt.Errorf("error checking package status: %w", err)
	}

	return p.diff(installed)
}

// diff compares the installation status to the desired state.
func (p *Package) diff(installed bool) (bool, tensile.Diff, error) {
	current := tensile.Absent
	if installed {
		current = tensile.Present
	}
	if current == p.desired() {
		return false, nil, nil
	}
	return true, diff.NewFieldChanges(&diff.FieldChange{
		Field: "state",
		Old:   string(current),
		New:   string(p.desired()),
	}), nil
}

// Execute implements [tensile.Executor].
func (p *Package) Execute(c tensile.Wire) (tensile.Diff, error) {
	mgr, err := p.manager(c)
	if err != nil {
		return nil, err
	}

	switch p.desired() { //nolint:exhaustive // ReadOnly is unsupported, rejected by Validate
	case tensile.Present:
		if err := mgr.Install(c.Context(), p.Name); err != nil {
			return nil, fmt.Errorf("error installing package: %w", err)
		}
		return nil, nil //nolint:nilnil // nil Diff is valid
	case tensile.Absent:
		if err := mgr.Remove(c.Context(), p.Name); err != nil {
			return nil, fmt.Errorf("error removing package: %w", err)
		}
		return nil, nil //nolint:nilnil // nil Diff is valid
	default:
		return nil, fmt.Errorf("unhandled desired state: %q", p.desired())
	}
}

func (p *Package) desired() tensile.State {
	return p.State.OrDefault()
}

func (p *Package) manager(c tensile.Wire) (PackageManager, error) {
	if p.Manager != "" {
		return PackageManagerByName(p.Manager)
	}
	return DetectPackageManager(c.Context(), p.Name)
}
