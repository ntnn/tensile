package std

import (
	"context"
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

var serviceManagers = registry[ServiceManager]{kind: "service manager"}

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
