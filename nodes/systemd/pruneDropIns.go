package systemd

import (
	"errors"
	"path/filepath"

	"github.com/ntnn/tensile"
	"github.com/ntnn/tensile/nodes/std"
)

var (
	_ tensile.Identifier = (*PruneDropIns)(nil)
	_ tensile.Validator  = (*PruneDropIns)(nil)
	_ tensile.Depender   = (*PruneDropIns)(nil)
	_ tensile.Notifier   = (*PruneDropIns)(nil)
	_ tensile.Executor   = (*PruneDropIns)(nil)
)

// PruneDropInsIdentity returns the identity of the node pruning the drop-in directory at path.
func PruneDropInsIdentity(path string) tensile.Identity {
	return tensile.AsIdentity("systemdPruneDropIns", "path", path)
}

// PruneDropIns removes all entries of a unit's drop-in directory not managed by a [Unit].
// This includes any file, not just unit files..
type PruneDropIns struct {
	// Name is the unit file name, e.g. foo.service.
	Name string

	// Dir defaults to [DefaultDir].
	Dir string
}

// Path returns the path of the drop-in directory.
func (p *PruneDropIns) Path() string {
	dir := p.Dir
	if dir == "" {
		dir = DefaultDir
	}
	return filepath.Join(dir, p.Name+".d")
}

// Identity implements [tensile.Identifier].
func (p *PruneDropIns) Identity() tensile.Identity {
	return PruneDropInsIdentity(p.Path())
}

// Validate implements [tensile.Validator].
func (p *PruneDropIns) Validate(_ tensile.Wire) error {
	if p.Name == "" {
		return errors.New("name is required")
	}
	return nil
}

// DependsOn implements [tensile.Depender].
func (p *PruneDropIns) DependsOn() ([]tensile.Identity, error) {
	return p.prune().DependsOn() //nolint:wrapcheck // std.PruneUnmanagedFiles names the operation
}

// Notifies implements [tensile.Notifier].
func (p *PruneDropIns) Notifies() ([]tensile.Identity, error) {
	return []tensile.Identity{DaemonReloadIdentity()}, nil
}

// NeedsExecution implements [tensile.Executor].
func (p *PruneDropIns) NeedsExecution(wire tensile.Wire) (bool, tensile.Diff, error) {
	return p.prune().NeedsExecution(wire) //nolint:wrapcheck // std.PruneUnmanagedFiles names the operation
}

// Execute implements [tensile.Executor].
func (p *PruneDropIns) Execute(wire tensile.Wire) (tensile.Diff, error) {
	return p.prune().Execute(wire) //nolint:wrapcheck // std.PruneUnmanagedFiles names the operation
}

// prune returns the node pruning the drop-in directory.
func (p *PruneDropIns) prune() *std.PruneUnmanagedFiles {
	return &std.PruneUnmanagedFiles{Dir: p.Path()}
}
