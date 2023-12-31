package nodes

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"

	"github.com/ntnn/tensile"
)

var _ tensile.Node = (*Link)(nil)

type LinkType string

const (
	Softlink LinkType = "soft"
	Hardlink LinkType = "hard"
)

type Link struct {
	Target string
	Source string
	Type   LinkType

	parentDirs *ParentDirs
}

func (link *Link) Validate() error {
	if link.Target == "" {
		return fmt.Errorf("target cannot be empty")
	}

	if link.Type == "" {
		link.Type = Softlink
	}

	if link.parentDirs == nil {
		link.parentDirs = NewParentDirs(link.Target)
	}

	return nil
}

func (link Link) Shape() tensile.Shape {
	return tensile.Path
}

func (link Link) Identifier() string {
	return link.Target
}

var _ tensile.AfterNoder = (*Link)(nil)

func (link Link) AfterNodes() []string {
	return append(
		link.parentDirs.AfterNodes(),
		link.Source,
	)
}

func (link Link) NeedsExecution(ctx tensile.Context) (bool, error) {
	sourceInfo, err := os.Lstat(link.Source)
	if err != nil && !os.IsNotExist(err) {
		// TODO should an error be thrown if the source does not exist?
		// technically yes but the source could be created during a run
		// and symlink do not have to be valid
		return false, fmt.Errorf("error reading info of source path %q: %w", link.Source, err)
	}

	targetInfo, err := os.Lstat(link.Target)
	if err != nil {
		// return execution need if the target does not exist
		if os.IsNotExist(err) {
			return true, nil
		}
		return false, fmt.Errorf("error reading info of target path %q: %w", link.Target, err)
	}

	if link.Type == Hardlink {
		return !os.SameFile(sourceInfo, targetInfo), nil
	}

	if targetInfo.Mode()&fs.ModeSymlink != fs.ModeSymlink {
		return true, nil
	}

	s, err := os.Readlink(link.Target)
	if err != nil {
		return false, fmt.Errorf("nodes: error reading target of symbolic link: %w", err)
	}

	return filepath.Clean(link.Source) != s, nil
}

func (link Link) Execute(ctx tensile.Context) (any, error) {
	if err := os.Remove(link.Target); err != nil && !os.IsNotExist(err) {
		return nil, fmt.Errorf("error removing target %q: %w", link.Target, err)
	}

	switch link.Type {
	case Softlink:
		return nil, os.Symlink(link.Source, link.Target)
	case Hardlink:
		return nil, os.Link(link.Source, link.Target)
	default:
		return nil, fmt.Errorf("unknown link type: %q", link.Type)
	}
}
