package std

import (
	"os/exec"

	"github.com/ntnn/tensile"
)

var (
	_ tensile.Identifier = (*SystemdDaemonReload)(nil)
	_ tensile.Executor   = (*SystemdDaemonReload)(nil)
)

// SystemdDaemonReloadIdentity returns the identity of the node reloading systemd.
func SystemdDaemonReloadIdentity() tensile.Identity {
	return tensile.AsIdentity("systemdDaemonReload")
}

// SystemdDaemonReload reloads systemd unit files and drop-ins.
//
// This node should only be used as a [tensile.Handler] to trigger
// a reload after a unit file or drop-in was changed.
//
// If enqueued as a node it _will_ reload systemd on every run.
type SystemdDaemonReload struct{}

// Identity implements [tensile.Identifier].
func (s *SystemdDaemonReload) Identity() tensile.Identity {
	return SystemdDaemonReloadIdentity()
}

// NeedsExecution implements [tensile.Executor].
func (s *SystemdDaemonReload) NeedsExecution(_ tensile.Wire) (bool, tensile.Diff, error) {
	return true, nil, nil
}

// Execute implements [tensile.Executor].
func (s *SystemdDaemonReload) Execute(wire tensile.Wire) (tensile.Diff, error) {
	_, err := exec.CommandContext(wire.Context(), "systemctl", "daemon-reload").CombinedOutput()
	if err != nil {
		return nil, err
	}
	return nil, nil //nolint:nilnil // nil Diff is valid
}
