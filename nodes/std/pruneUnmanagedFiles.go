package std

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/ntnn/tensile"
	"github.com/ntnn/tensile/pkg/diff"
)

var _ tensile.Identifier = (*PruneUnmanagedFiles)(nil)
var _ tensile.Validator = (*PruneUnmanagedFiles)(nil)
var _ tensile.Depender = (*PruneUnmanagedFiles)(nil)
var _ tensile.Executor = (*PruneUnmanagedFiles)(nil)

// PruneUnmanagedFilesIdentity returns the identity of the node pruning unmanaged files in dir.
func PruneUnmanagedFilesIdentity(dir string) tensile.Identity {
	return tensile.AsIdentity("pruneUnmanagedFiles", "dir", dir)
}

// PruneUnmanagedFiles removes direct entries of Dir whose [FileIdentity] is not claimed by any node.
// Entries are removed recursively, an unmanaged directory is removed with its contents.
type PruneUnmanagedFiles struct {
	Dir string
}

// Identity implements [tensile.Identifier].
func (p *PruneUnmanagedFiles) Identity() tensile.Identity {
	return PruneUnmanagedFilesIdentity(p.Dir)
}

// Validate implements [tensile.Validator].
func (p *PruneUnmanagedFiles) Validate(_ tensile.Wire) error {
	if p.Dir == "" {
		return errors.New("dir is required")
	}
	return nil
}

// DependsOn implements [tensile.Depender].
func (p *PruneUnmanagedFiles) DependsOn() ([]tensile.Identity, error) {
	return append(
		ParentDirIdentities(p.Dir),
		FileIdentity(p.Dir),
	), nil
}

// NeedsExecution implements [tensile.Executor].
func (p *PruneUnmanagedFiles) NeedsExecution(wire tensile.Wire) (bool, tensile.Diff, error) {
	unmanaged, err := p.unmanaged(wire)
	if err != nil {
		return false, nil, err
	}
	if len(unmanaged) == 0 {
		return false, nil, nil
	}

	changes := make([]*diff.FieldChange, len(unmanaged))
	for i, path := range unmanaged {
		changes[i] = &diff.FieldChange{Field: path, Old: path, New: diff.Absent}
	}
	return true, diff.NewFieldChanges(changes...), nil
}

// Execute implements [tensile.Executor].
func (p *PruneUnmanagedFiles) Execute(wire tensile.Wire) (tensile.Diff, error) {
	unmanaged, err := p.unmanaged(wire)
	if err != nil {
		return nil, err
	}
	for _, path := range unmanaged {
		if err := os.RemoveAll(path); err != nil {
			return nil, fmt.Errorf("removing unmanaged file: %w", err)
		}
	}
	return nil, nil //nolint:nilnil // nil Diff is valid
}

// unmanaged returns the paths of direct entries of Dir whose [FileIdentity] is not claimed.
// A missing Dir yields no paths.
func (p *PruneUnmanagedFiles) unmanaged(wire tensile.Wire) ([]string, error) {
	topo, err := wire.Topology()
	if err != nil {
		return nil, fmt.Errorf("getting topology: %w", err)
	}

	entries, err := os.ReadDir(p.Dir)
	if os.IsNotExist(err) {
		// dir does not exist, nothing to prune
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("reading directory: %w", err)
	}

	unmanaged := []string{}
	for _, entry := range entries {
		path := filepath.Join(p.Dir, entry.Name())
		if _, claimed := topo.Claimer(FileIdentity(path)); claimed {
			continue
		}
		unmanaged = append(unmanaged, path)
	}
	return unmanaged, nil
}
