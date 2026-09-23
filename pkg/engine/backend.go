package engine

import (
	"errors"
	"fmt"
	"slices"

	"github.com/ntnn/tensile"
	"github.com/ntnn/tensile/pkg/storage"
)

// scopedBackend restricts reads to the declared dependencies of a node.
type scopedBackend struct {
	backend storage.Backend[tensile.Identity]
	node    tensile.Identity
	deps    []tensile.Identity
}

var _ storage.Backend[tensile.Identity] = (*scopedBackend)(nil)

// Load implements [storage.Backend].
func (s *scopedBackend) Load(key tensile.Identity) (any, error) {
	if !slices.Contains(s.deps, key) {
		return nil, fmt.Errorf("%s is not a dependency of %s", key, s.node)
	}
	return s.backend.Load(key)
}

// Store implements [storage.Backend]. Nodes cannot store outputs directly.
func (s *scopedBackend) Store(tensile.Identity, any) error {
	return errors.New("store is read-only")
}
