package std

import (
	"errors"
	"fmt"

	"github.com/ntnn/tensile"
)

var _ tensile.Identifier = (*Service)(nil)
var _ tensile.Validator = (*Service)(nil)
var _ tensile.Provider = (*Service)(nil)
var _ tensile.Executor = (*Service)(nil)

// ServiceIdentity returns the identity of the node managing the named service.
func ServiceIdentity(name string) tensile.Identity {
	return tensile.AsIdentity("service", "name", name)
}

// Service ensures a service is in the desired state.
// Nil means the state is unmanaged.
type Service struct {
	Name    string
	Enabled *bool
	Running *bool
	// Manager selects a registered [ServiceManager] by name.
	// Empty means detection.
	Manager string
}

// Validate implements [tensile.Validator].
func (s *Service) Validate(_ tensile.Wire) error {
	if s.Name == "" {
		return errors.New("name is required")
	}
	if s.Enabled == nil && s.Running == nil {
		return errors.New("at least one of Enabled and Running is required")
	}
	return nil
}

// Identity implements [tensile.Identifier].
func (s *Service) Identity() tensile.Identity {
	return ServiceIdentity(s.Name)
}

// Provides implements [tensile.Provider].
func (s *Service) Provides() ([]tensile.Identity, error) {
	return []tensile.Identity{ServiceIdentity(s.Name)}, nil
}

// NeedsExecution implements [tensile.Executor].
func (s *Service) NeedsExecution(c tensile.Wire) (bool, error) {
	mgr, err := s.manager(c)
	if err != nil {
		return false, err
	}

	status, err := mgr.Status(c.Context(), s.Name)
	if err != nil {
		return false, fmt.Errorf("error checking service status: %w", err)
	}

	if s.Enabled != nil && status.Enabled != *s.Enabled {
		return true, nil
	}
	if s.Running != nil && status.Active != *s.Running {
		return true, nil
	}
	return false, nil
}

// Execute implements [tensile.Executor].
func (s *Service) Execute(c tensile.Wire) error {
	mgr, err := s.manager(c)
	if err != nil {
		return err
	}

	// unmanaged fields keep their current state
	desired, err := mgr.Status(c.Context(), s.Name)
	if err != nil {
		return fmt.Errorf("error checking service status: %w", err)
	}
	if s.Enabled != nil {
		desired.Enabled = *s.Enabled
	}
	if s.Running != nil {
		desired.Active = *s.Running
	}

	if err := mgr.Apply(c.Context(), s.Name, desired); err != nil {
		return fmt.Errorf("error applying service status: %w", err)
	}
	return nil
}

func (s *Service) manager(c tensile.Wire) (ServiceManager, error) {
	if s.Manager != "" {
		return ServiceManagerByName(s.Manager)
	}
	return DetectServiceManager(c.Context(), s.Name)
}
