package nodes

import (
	"path/filepath"

	"github.com/ntnn/tensile"
)

type ParentDirs struct {
	ParentDirs []string
}

func NewParentDirs(path string) *ParentDirs {
	d := new(ParentDirs)
	d.ParentDirs = walkDirs(path)
	return d
}

var _ tensile.AfterNoder = (*ParentDirs)(nil)

func (parent ParentDirs) AfterNodes() []string {
	return parent.ParentDirs
}

func walkDirs(target string) []string {
	last := filepath.FromSlash(target)
	ret := []string{}

	for s := filepath.Dir(target); s != last; s = filepath.Dir(s) {
		// 1. target=/ -> no dependency
		// 2. target=/a -> []string{"/"}
		if s == target || s == last || s == filepath.VolumeName(s) {
			break
		}
		ret = append(ret, tensile.FormatIdentity(tensile.Path, s))
		last = s
	}

	return ret
}
