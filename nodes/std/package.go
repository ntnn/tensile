package std

import (
	"errors"
	"fmt"

	"github.com/ntnn/tensile"
)

var _ tensile.Identifier = (*Package)(nil)
var _ tensile.Validator = (*Package)(nil)
var _ tensile.Depender = (*Package)(nil)
var _ tensile.Executor = (*Package)(nil)

// PackageIdentity returns the identity of the node managing the named package.
func PackageIdentity(name string) tensile.Identity {
	return tensile.AsIdentity("package", "name", name)
}

// PackageState is the desired presence of a package.
type PackageState string

// Desired presence states for [Package].
const (
	PackagePresent PackageState = "present"
	PackageAbsent  PackageState = "absent"
)

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
	// Empty means [PackagePresent].
	State PackageState
	// Manager selects a registered [PackageManager] by name.
	// Empty means detection.
	Manager string
}

// Validate implements [tensile.Validator].
func (p *Package) Validate(_ tensile.Wire) error {
	if p.Name == "" {
		return errors.New("name is required")
	}
	switch p.State {
	case "", PackagePresent, PackageAbsent:
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

// NeedsExecution implements [tensile.Executor].
func (p *Package) NeedsExecution(c tensile.Wire) (bool, error) {
	mgr, err := p.manager(c)
	if err != nil {
		return false, err
	}

	installed, err := mgr.Installed(c.Context(), p.Name)
	if err != nil {
		return false, fmt.Errorf("error checking package status: %w", err)
	}

	return installed != (p.desired() == PackagePresent), nil
}

// Execute implements [tensile.Executor].
func (p *Package) Execute(c tensile.Wire) error {
	mgr, err := p.manager(c)
	if err != nil {
		return err
	}

	switch p.desired() {
	case PackagePresent:
		if err := mgr.Install(c.Context(), p.Name); err != nil {
			return fmt.Errorf("error installing package: %w", err)
		}
		return nil
	case PackageAbsent:
		if err := mgr.Remove(c.Context(), p.Name); err != nil {
			return fmt.Errorf("error removing package: %w", err)
		}
		return nil
	default:
		return fmt.Errorf("unhandled desired state: %q", p.desired())
	}
}

func (p *Package) desired() PackageState {
	if p.State == "" {
		return PackagePresent
	}
	return p.State
}

func (p *Package) manager(c tensile.Wire) (PackageManager, error) {
	if p.Manager != "" {
		return PackageManagerByName(p.Manager)
	}
	return DetectPackageManager(c.Context(), p.Name)
}
