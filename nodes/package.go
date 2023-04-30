package nodes

import (
	"fmt"

	"github.com/ntnn/tensile"
)

type PackageState int

const (
	Undefined PackageState = iota
	Installed
	Removed
	Latest
)

var DefaultPackageState = Installed

var _ tensile.Node = (*Package)(nil)

type Package struct {
	State    PackageState
	Name     string
	Provider string
}

func (pack Package) Shape() tensile.Shape {
	return tensile.Package
}

func (pack Package) Identifier() string {
	return pack.Name
}

func (pack *Package) Validate() error {
	if pack.State < Installed || pack.State > Latest {
		pack.State = DefaultPackageState
	}

	if pack.Name == "" {
		return fmt.Errorf("nodes: package name cannot be empty")
	}

	if pack.Provider == "" {
		// TODO choose provider
	}

	return nil
}

func (pack Package) Execute(ctx tensile.Context) (any, error) {
	// TODO
	return nil, nil
}
