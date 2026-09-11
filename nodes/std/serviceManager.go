package std

import (
	"context"
	"fmt"
	"slices"
	"sync"
)

// ServiceStatus is the observable state of a service.
type ServiceStatus struct {
	Enabled bool
	Active  bool
}

// ServiceManager queries and changes service state.
type ServiceManager interface {
	// Handles reports whether this manager handles the service.
	Handles(ctx context.Context, name string) (bool, error)
	// Status returns the current state of the service.
	Status(ctx context.Context, name string) (ServiceStatus, error)
	// Apply transitions the service to the desired state.
	Apply(ctx context.Context, name string, desired ServiceStatus) error
}

type serviceManagerRegistry struct {
	mu      sync.RWMutex
	entries map[string]ServiceManager
	order   []string
}

func (r *serviceManagerRegistry) register(name string, manager ServiceManager) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, ok := r.entries[name]; ok {
		panic(fmt.Sprintf("service manager %q already registered", name))
	}
	if r.entries == nil {
		r.entries = map[string]ServiceManager{}
	}
	r.entries[name] = manager
	r.order = append(r.order, name)
}

func (r *serviceManagerRegistry) byName(name string) (ServiceManager, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	manager, ok := r.entries[name]
	if !ok {
		return nil, fmt.Errorf("no service manager registered under %q", name)
	}
	return manager, nil
}

// detect walks managers in reverse registration order so a later registration wins over an earlier one.
func (r *serviceManagerRegistry) detect(ctx context.Context, name string) (ServiceManager, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	for _, entryName := range slices.Backward(r.order) {
		manager := r.entries[entryName]
		handles, err := manager.Handles(ctx, name)
		if err != nil {
			return nil, fmt.Errorf("error checking if %q handles %q: %w", entryName, name, err)
		}
		if handles {
			return manager, nil
		}
	}
	return nil, fmt.Errorf("no registered service manager handles %q", name)
}

var serviceManagers = serviceManagerRegistry{
	entries: map[string]ServiceManager{},
	order:   []string{},
}

// RegisterServiceManager registers a service manager.
// Panics when name is already registered.
func RegisterServiceManager(name string, manager ServiceManager) {
	serviceManagers.register(name, manager)
}

// ServiceManagerByName returns the service manager registered under name.
func ServiceManagerByName(name string) (ServiceManager, error) {
	return serviceManagers.byName(name)
}

// DetectServiceManager returns the last registered service manager
// that handles the named service.
func DetectServiceManager(ctx context.Context, name string) (ServiceManager, error) {
	return serviceManagers.detect(ctx, name)
}
