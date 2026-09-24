package std

import (
	"errors"
	"fmt"

	"github.com/ntnn/tensile"
)

var (
	_ tensile.Identifier = (*PackageManagerUpdate)(nil)
	_ tensile.Executor   = (*PackageManagerUpdate)(nil)
)

// PackageManagerUpdateIdentity returns the identity of the node updating package lists.
func PackageManagerUpdateIdentity() tensile.Identity {
	return tensile.AsIdentity("packageManagerUpdate")
}

// PackageManagerUpdate refreshes the package lists of all registered package managers.
// It does not upgrade packages.
type PackageManagerUpdate struct{}

// Identity implements [tensile.Identifier].
func (p *PackageManagerUpdate) Identity() tensile.Identity {
	return PackageManagerUpdateIdentity()
}

// NeedsExecution implements [tensile.Executor].
// List staleness is not observable, so it always executes.
func (p *PackageManagerUpdate) NeedsExecution(_ tensile.Wire) (bool, tensile.Diff, error) {
	return true, nil, nil
}

// Execute implements [tensile.Executor].
func (p *PackageManagerUpdate) Execute(c tensile.Wire) (tensile.Diff, error) {
	var errs []error
	for name, manager := range packageManagers.all() {
		if err := manager.Update(c.Context()); err != nil {
			errs = append(errs, fmt.Errorf("updating %q: %w", name, err))
		}
	}
	return nil, errors.Join(errs...)
}
