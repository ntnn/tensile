package diff

import (
	"log/slog"

	"github.com/aymanbagabas/go-udiff"
	"github.com/ntnn/tensile"
)

var _ tensile.Diff = Unified{}

// Unified is a line-based unified [tensile.Diff] between Old and New content.
// It uses [udiff.Unified].
type Unified struct {
	// Path labels both sides of the diff.
	Path string
	Old  string
	New  string
}

// String renders the unified diff.
func (u Unified) String() string {
	return udiff.Unified(u.Path, u.Path, u.Old, u.New)
}

// LogValue renders the path and the unified diff.
func (u Unified) LogValue() slog.Value {
	return slog.GroupValue(
		slog.String("path", u.Path),
		slog.String("diff", u.String()),
	)
}
