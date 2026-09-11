package std

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"strings"
)

var _ ServiceManager = (*Systemd)(nil)

func init() {
	RegisterServiceManager("systemd", &Systemd{})
}

// Systemd manages services through systemctl.
type Systemd struct {
	// run executes systemctl with args and returns combined output.
	run func(ctx context.Context, args ...string) ([]byte, error)
	// available reports whether systemd is present.
	available func() bool
}

// Handles implements [ServiceManager].
func (systemd *Systemd) Handles(ctx context.Context, name string) (bool, error) {
	if !systemd.isAvailable() {
		return false, nil
	}
	props, err := systemd.show(ctx, name)
	if err != nil {
		return false, err
	}
	return props["LoadState"] != "not-found", nil
}

// Status implements [ServiceManager].
func (systemd *Systemd) Status(ctx context.Context, name string) (ServiceStatus, error) {
	var status ServiceStatus

	props, err := systemd.show(ctx, name)
	if err != nil {
		return status, err
	}

	switch loadState := props["LoadState"]; loadState {
	case "loaded", "masked":
	default:
		return status, fmt.Errorf("unit %q has load state %q", name, loadState)
	}

	unitFileState := props["UnitFileState"]
	status.Enabled = unitFileState == "enabled" || unitFileState == "enabled-runtime"
	status.Active = props["ActiveState"] == "active"

	return status, nil
}

// Apply implements [ServiceManager].
func (systemd *Systemd) Apply(ctx context.Context, name string, desired ServiceStatus) error {
	current, err := systemd.Status(ctx, name)
	if err != nil {
		return err
	}

	if current.Enabled != desired.Enabled {
		verb := "enable"
		if !desired.Enabled {
			verb = "disable"
		}
		if err := systemd.transition(ctx, verb, name); err != nil {
			return err
		}
	}

	if current.Active != desired.Active {
		verb := "start"
		if !desired.Active {
			verb = "stop"
		}
		if err := systemd.transition(ctx, verb, name); err != nil {
			return err
		}
	}

	return nil
}

// show queries unit properties as Key=value pairs.
func (systemd *Systemd) show(ctx context.Context, name string) (map[string]string, error) {
	out, err := systemd.systemctl(ctx,
		"show", "--property=LoadState,UnitFileState,ActiveState", "--", name)
	if err != nil {
		return nil, fmt.Errorf("running systemctl show: %w", err)
	}

	props := map[string]string{}
	for line := range strings.Lines(string(out)) {
		key, value, ok := strings.Cut(strings.TrimSpace(line), "=")
		if !ok {
			continue
		}
		props[key] = value
	}
	return props, nil
}

func (systemd *Systemd) transition(ctx context.Context, verb, name string) error {
	out, err := systemd.systemctl(ctx, verb, "--", name)
	if err != nil {
		return fmt.Errorf("systemctl %s %q: %w: %s", verb, name, err, strings.TrimSpace(string(out)))
	}
	return nil
}

func (systemd *Systemd) systemctl(ctx context.Context, args ...string) ([]byte, error) {
	if systemd.run != nil {
		return systemd.run(ctx, args...)
	}
	//nolint:gosec // args are fixed verbs plus the unit name
	return exec.CommandContext(ctx, "systemctl", args...).CombinedOutput()
}

func (systemd *Systemd) isAvailable() bool {
	if systemd.available != nil {
		return systemd.available()
	}
	if _, err := os.Stat("/run/systemd/system"); err == nil {
		return true
	}
	_, err := exec.LookPath("systemctl")
	return err == nil
}
