package systemd

import (
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/coreos/go-systemd/v22/unit"
	"github.com/ntnn/tensile"
	"github.com/ntnn/tensile/nodes/std"
)

var (
	_ tensile.Identifier = (*Unit)(nil)
	_ tensile.Validator  = (*Unit)(nil)
	_ tensile.Conflictor = (*Unit)(nil)
	_ tensile.Depender   = (*Unit)(nil)
	_ tensile.Notifier   = (*Unit)(nil)
	_ tensile.Executor   = (*Unit)(nil)
)

// DefaultDir is the directory for unit files and drop-ins managed by the administrator.
const DefaultDir = "/etc/systemd/system"

// UnitIdentity returns the identity of the node managing the unit file or drop-in at path.
func UnitIdentity(path string) tensile.Identity {
	return tensile.AsIdentity("systemdUnit", "path", path)
}

// Unit ensures a unit file or drop-in has exactly the declared sections.
type Unit struct {
	// State is the desired state.
	// Defaults to [tensile.Present].
	State tensile.State

	// Name is the unit file name, e.g. foo.service.
	Name string

	// DropIn is the drop-in name without .conf.
	// Empty manages the unit file itself.
	DropIn string

	// Dir defaults to [DefaultDir].
	Dir string

	Sections []*unit.UnitSection
}

// Path returns the path of the unit file or drop-in.
func (u *Unit) Path() string {
	dir := u.Dir
	if dir == "" {
		dir = DefaultDir
	}
	if u.DropIn == "" {
		return filepath.Join(dir, u.Name)
	}
	return filepath.Join(dir, u.Name+".d", u.DropIn+".conf")
}

// Identity implements [tensile.Identifier].
func (u *Unit) Identity() tensile.Identity {
	return UnitIdentity(u.Path())
}

// Validate implements [tensile.Validator].
func (u *Unit) Validate(_ tensile.Wire) error {
	if u.Name == "" {
		return errors.New("name is required")
	}
	return u.State.Valid() //nolint:wrapcheck // names the state
}

// Conflicts implements [tensile.Conflictor].
func (u *Unit) Conflicts() ([]tensile.Identity, error) {
	return []tensile.Identity{std.FileIdentity(u.Path())}, nil
}

// DependsOn implements [tensile.Depender].
func (u *Unit) DependsOn() ([]tensile.Identity, error) {
	return std.ParentDirIdentities(u.Path()), nil
}

// Notifies implements [tensile.Notifier].
func (u *Unit) Notifies() ([]tensile.Identity, error) {
	return []tensile.Identity{DaemonReloadIdentity()}, nil
}

// NeedsExecution implements [tensile.Executor].
func (u *Unit) NeedsExecution(wire tensile.Wire) (bool, tensile.Diff, error) {
	switch u.State.OrDefault() {
	case tensile.ReadOnly:
		return false, nil, nil
	case tensile.Absent:
		return u.file().NeedsExecution(wire) //nolint:wrapcheck // std.File names the operation
	case tensile.Present:
		content, err := u.content()
		if err != nil {
			return false, nil, err
		}
		return content.NeedsExecution(wire) //nolint:wrapcheck // std.FileContent names the operation
	default:
		return false, nil, fmt.Errorf("unknown state %q", u.State)
	}
}

// Execute implements [tensile.Executor].
func (u *Unit) Execute(wire tensile.Wire) (tensile.Diff, error) {
	switch u.State.OrDefault() {
	case tensile.ReadOnly:
		return nil, nil //nolint:nilnil // nil Diff is valid
	case tensile.Absent:
		return u.file().Execute(wire) //nolint:wrapcheck
	case tensile.Present:
		return u.write(wire)
	default:
		return nil, fmt.Errorf("unknown state %q", u.State)
	}
}

// file returns the node managing the existence of the file.
func (u *Unit) file() *std.File {
	return &std.File{
		Path:  u.Path(),
		State: u.State,
	}
}

// content returns the node managing the rendered sections.
func (u *Unit) content() (*std.FileContent, error) {
	rendered, err := io.ReadAll(unit.SerializeSections(u.Sections))
	if err != nil {
		return nil, fmt.Errorf("serializing sections: %w", err)
	}

	return &std.FileContent{
		Path:    u.Path(),
		Content: string(rendered),
	}, nil
}

func (u *Unit) write(wire tensile.Wire) (tensile.Diff, error) {
	content, err := u.content()
	if err != nil {
		return nil, err
	}
	// in case the drop-in directory is not managed in tensile
	if err := os.MkdirAll(filepath.Dir(u.Path()), std.DefaultDirMode); err != nil {
		return nil, fmt.Errorf("creating parent directory: %w", err)
	}
	// create the file, FileContent doesn't
	if _, err := u.file().Execute(wire); err != nil {
		return nil, err //nolint:wrapcheck // std.File names the operation
	}
	return content.Execute(wire) //nolint:wrapcheck // std.FileContent names the operation
}
