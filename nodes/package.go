package nodes

import (
	"fmt"

	"github.com/ntnn/tensile"
)

type PackageState string

const (
	Undefined   PackageState = ""
	Installed   PackageState = "installed"
	Uninstalled PackageState = "uninstalled"
	Latest      PackageState = "latest"
)

type ChecksumType int

const (
	Md5 ChecksumType = iota + 1
)

var _ tensile.Node = (*Package)(nil)

type Package struct {
	Name string
	// State defaults to latest.
	State PackageState

	// NameByOS allows to set a name depending on the OS.
	// If no name is set for a provider Package.Name is used.
	//
	// p.Name = "netcat"
	// p.NameByOS = map[string]string{
	//   "SLES": "gnu-netcat" // or netcat-openbsd
	//   "RHEL": "nmap" // netcat is part of nmap
	// }
	NameByOS map[string]string

	// If source URI is set the file will be downloaded first if
	// necessary. The name should still be the name as it will be
	// recognized by the package manager on the system.
	// Examples:
	//   file:///tmp/downloaded.rpm
	//   https://central-repository/downloaded.rpm
	SourceURI string

	// If ChecksumSourceURI is set the file downloaded from SourceURI will
	// be validated against this hash using ChecksumType at this source.
	//
	// The file should have the usual checksum layout:
	// <checksum> <file>
	ChecksumType      ChecksumType
	ChecksumSourceURI string
}

func (p Package) Identity() (tensile.Shape, string) {
	return tensile.Package, p.Name
}

func (p Package) IsCollision(other any) error {
	otherP, ok := other.(*Package)
	if !ok {
		return fmt.Errorf("other is not %T, cannot compare", p)
	}

	if p.State != otherP.State {
		return fmt.Errorf("conflicting states %q and %q", p.State, otherP.State)
	}

	return nil
}

func (p *Package) Validate() error {
	if p.Name == "" {
		return fmt.Errorf("package name can not be empty")
	}

	if p.State == Undefined {
		p.State = Latest
	}

	return nil
}

func (p *Package) Execute(ctx tensile.Context) (any, error) {
	return nil, nil
}
