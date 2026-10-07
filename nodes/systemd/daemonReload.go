package systemd

import (
	"os/exec"

	"github.com/ntnn/tensile"
)

var (
	_ tensile.Identifier = (*DaemonReload)(nil)
	_ tensile.Executor   = (*DaemonReload)(nil)
)

// DaemonReloadIdentity returns the identity of the node reloading systemd.
func DaemonReloadIdentity() tensile.Identity {
	return tensile.AsIdentity("systemdDaemonReload")
}

// DaemonReload reloads systemd unit files and drop-ins.
//
// This node should only be used as a [tensile.Handler] to trigger
// a reload after a unit file or drop-in was changed.
//
// If enqueued as a node it _will_ reload systemd on every run.
type DaemonReload struct{}

// Identity implements [tensile.Identifier].
func (d *DaemonReload) Identity() tensile.Identity {
	return DaemonReloadIdentity()
}

// NeedsExecution implements [tensile.Executor].
func (d *DaemonReload) NeedsExecution(_ tensile.Wire) (bool, tensile.Diff, error) {
	return true, nil, nil
}

// Execute implements [tensile.Executor].
func (d *DaemonReload) Execute(wire tensile.Wire) (tensile.Diff, error) {
	_, err := exec.CommandContext(wire.Context(), "systemctl", "daemon-reload").CombinedOutput()
	if err != nil {
		return nil, err
	}
	return nil, nil //nolint:nilnil // nil Diff is valid
}
