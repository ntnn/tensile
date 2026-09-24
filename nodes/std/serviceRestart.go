package std

import (
	"errors"
	"fmt"

	"github.com/ntnn/tensile"
)

var (
	_ tensile.Identifier = (*ServiceRestart)(nil)
	_ tensile.Validator  = (*ServiceRestart)(nil)
	_ tensile.Executor   = (*ServiceRestart)(nil)
)

// ServiceRestartIdentity returns the identity of the node restarting the named service.
func ServiceRestartIdentity(name string) tensile.Identity {
	return tensile.AsIdentity("serviceRestart", "name", name)
}

// ServiceRestart restarts a service.
//
// This node should only be used as a [tensile.Handler] to trigger
// a service restart after a configuration for this service was changed
// and the service does not reload dynamically.
//
// If enqueued as a node it _will_ restart the service on every run.
type ServiceRestart struct {
	Name string
	// Manager selects a registered [ServiceManager] by name.
	// Empty means detection.
	Manager string
}

// Validate implements [tensile.Validator].
func (s *ServiceRestart) Validate(_ tensile.Wire) error {
	if s.Name == "" {
		return errors.New("name is required")
	}
	return nil
}

// Identity implements [tensile.Identifier].
func (s *ServiceRestart) Identity() tensile.Identity {
	return ServiceRestartIdentity(s.Name)
}

// NeedsExecution implements [tensile.Executor].
func (s *ServiceRestart) NeedsExecution(_ tensile.Wire) (bool, tensile.Diff, error) {
	return true, nil, nil
}

// Execute implements [tensile.Executor].
func (s *ServiceRestart) Execute(c tensile.Wire) (tensile.Diff, error) {
	mgr, err := s.manager(c)
	if err != nil {
		return nil, err
	}
	if err := mgr.Restart(c.Context(), s.Name); err != nil {
		return nil, fmt.Errorf("error restarting service: %w", err)
	}
	return nil, nil //nolint:nilnil // nil Diff is valid
}

func (s *ServiceRestart) manager(c tensile.Wire) (ServiceManager, error) {
	if s.Manager != "" {
		return ServiceManagerByName(s.Manager)
	}
	return DetectServiceManager(c.Context(), s.Name)
}
