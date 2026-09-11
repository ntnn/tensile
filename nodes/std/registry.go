package std

import (
	"context"
	"fmt"
	"slices"
	"sync"
)

// handler is the detection seam shared by manager registries.
type handler interface {
	// Handles reports whether this manager handles the named resource.
	Handles(ctx context.Context, name string) (bool, error)
}

// registry stores named managers.
// kind names the manager type in error and panic messages.
type registry[T handler] struct {
	kind    string
	lock    sync.RWMutex
	entries map[string]T
	order   []string
}

func (r *registry[T]) register(name string, manager T) {
	r.lock.Lock()
	defer r.lock.Unlock()
	if _, ok := r.entries[name]; ok {
		panic(fmt.Sprintf("%s %q already registered", r.kind, name))
	}
	if r.entries == nil {
		r.entries = map[string]T{}
	}
	r.entries[name] = manager
	r.order = append(r.order, name)
}

func (r *registry[T]) byName(name string) (T, error) {
	r.lock.RLock()
	defer r.lock.RUnlock()
	manager, ok := r.entries[name]
	if !ok {
		var zero T
		return zero, fmt.Errorf("no %s registered under %q", r.kind, name)
	}
	return manager, nil
}

// detect walks managers in reverse registration order so a later registration wins over an earlier one.
func (r *registry[T]) detect(ctx context.Context, name string) (T, error) {
	r.lock.RLock()
	defer r.lock.RUnlock()
	var zero T
	for _, entryName := range slices.Backward(r.order) {
		manager := r.entries[entryName]
		handles, err := manager.Handles(ctx, name)
		if err != nil {
			return zero, fmt.Errorf("error checking if %q handles %q: %w", entryName, name, err)
		}
		if handles {
			return manager, nil
		}
	}
	return zero, fmt.Errorf("no registered %s handles %q", r.kind, name)
}
