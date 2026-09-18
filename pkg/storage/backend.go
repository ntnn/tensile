package storage

import (
	"fmt"
	"sync"
)

// Backend is the interface for what actually stores data retrievable from a [Store].
//
// This basically only exists to work around what Go allows for generics
// and interfaces:
//
// Interface methods in Go cannot have type parameters - only the
// interface itself can have type parameters.
//
// [tensile.Node] need a way to retrieve typed output from other nodes,
// due to the above a [tensile.Wire] cannot have a `Output(Identity) (T, error)`.
// So instead the wire has a `Store() storage.Store`, which then allows
// type safe access over [tensile.Identity] with a generic `Get` that
// type asserts the output.
//
// However there will be different implementations depending on the
// engine being used - so the [Store] needs this Backend.
//
// Basically the engine creates its Backend and stores node output in it
// and then creates a [Store] to pass as part of the [tensile.Wire] for
// nodes to retrieve data from the [Backend].
type Backend[K comparable] interface {
	Load(key K) (any, error)
	Store(key K, value any) error
}

// DefaultBackend is the default implementation of [Backend].
type DefaultBackend[K comparable] struct {
	lock   sync.RWMutex
	values map[K]any
}

// NewDefaultBackend returns an instantiated [DefaultBackend].
func NewDefaultBackend[K comparable]() *DefaultBackend[K] {
	return &DefaultBackend[K]{values: map[K]any{}}
}

// Load implements [Backend].
func (b *DefaultBackend[K]) Load(key K) (any, error) {
	b.lock.RLock()
	defer b.lock.RUnlock()
	v, ok := b.values[key]
	if !ok {
		return nil, fmt.Errorf("no value for key %v", key)
	}
	return v, nil
}

// Store implements [Backend].
func (b *DefaultBackend[K]) Store(key K, value any) error {
	b.lock.Lock()
	defer b.lock.Unlock()
	if _, ok := b.values[key]; ok {
		return fmt.Errorf("key %v already has a value", key)
	}
	b.values[key] = value
	return nil
}
