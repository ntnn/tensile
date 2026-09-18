package storage

import (
	"errors"
	"fmt"
)

// Store provides typed read access to keyed values.
// The zero value errors on every access.
type Store[K comparable] struct {
	backend Backend[K]
}

// New returns a Store reading through load.
func New[K comparable](backend Backend[K]) *Store[K] {
	return &Store[K]{
		backend: backend,
	}
}

// Get returns the value for key as T.
func (s *Store[K]) Get[T any](key K) (T, error) {
	var zero T

	if s.backend == nil {
		return zero, errors.New("store has no backend")
	}

	v, err := s.backend.Load(key)
	if err != nil {
		return zero, err
	}

	t, ok := v.(T)
	if !ok {
		return zero, fmt.Errorf("value for %v is %T, not %T", key, v, zero)
	}

	return t, nil
}
