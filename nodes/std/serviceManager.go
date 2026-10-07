package std

import (
	"context"
	"fmt"
	"time"
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
	// Restart restarts the service.
	Restart(ctx context.Context, name string) error
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

// DefaultServiceTimeout bounds waiting for a service manager to report
// the desired service state when no timeout is set.
const DefaultServiceTimeout = 30 * time.Second

// serviceWaitInterval is the delay between service status checks.
const serviceWaitInterval = 500 * time.Millisecond

// waitActive polls the service status until Active equals active.
// Zero timeout means [DefaultServiceTimeout].
func waitActive(ctx context.Context, mgr ServiceManager, name string, active bool, timeout time.Duration) error {
	if timeout == 0 {
		timeout = DefaultServiceTimeout
	}
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	ticker := time.NewTicker(serviceWaitInterval)
	defer ticker.Stop()

	for {
		status, err := mgr.Status(ctx, name)
		// the deadline can expire during a status check
		if ctx.Err() != nil {
			return fmt.Errorf("waiting for service %q to be active=%t: %w", name, active, ctx.Err())
		}
		if err != nil {
			return fmt.Errorf("error checking service status: %w", err)
		}
		if status.Active == active {
			return nil
		}

		select {
		case <-ctx.Done():
		case <-ticker.C:
		}
	}
}
