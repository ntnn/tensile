package e2e

import (
	"testing"

	"github.com/ntnn/tensile/test/e2e/framework"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSystemdUnit(t *testing.T) {
	t.Parallel()

	const dropInDir = "/etc/systemd/system/e2e-unit.service.d"

	env := framework.SharedContainer(t, framework.SystemdDebian, framework.Scenario{Name: "systemdunit"})

	exit, out := env.Exec(t, "sh", "-c",
		"mkdir -p "+dropInDir+" && printf '[Unit]\\nDescription=stray\\n' > "+dropInDir+"/20-stray.conf")
	require.Zero(t, exit, out)

	exit, out = env.RunScenario(t)
	require.Zero(t, exit, out)

	exit, out = env.Exec(t, "test", "-f", dropInDir+"/20-stray.conf")
	assert.NotZero(t, exit, "unmanaged drop-in must be pruned: %s", out)

	exit, out = env.Exec(t, "test", "-f", dropInDir+"/10-e2e.conf")
	assert.Zero(t, exit, "managed drop-in must survive pruning: %s", out)

	exit, out = env.Exec(t, "systemctl", "is-active", "e2e-unit.service")
	assert.Zero(t, exit, out)

	exit, out = env.Exec(t, "systemctl", "show", "-p", "Description", "e2e-unit.service")
	assert.Zero(t, exit, out)
	assert.Contains(t, out, "Description=tensile e2e drop-in\n", "drop-in must override the unit")

	exit, out = env.RunScenario(t)
	require.Zero(t, exit, out)
	assert.NotContains(t, out, "executed", "unchanged run must not execute any node")

	// the unit is loaded now, the new description is only visible after a daemon-reload
	exit, out = env.RunScenario(t, "-description", "tensile e2e changed")
	require.Zero(t, exit, out)
	assert.Contains(t, out, "handler executed", "changed drop-in must notify daemon-reload")

	exit, out = env.Exec(t, "systemctl", "show", "-p", "Description,NeedDaemonReload", "e2e-unit.service")
	assert.Zero(t, exit, out)
	assert.Contains(t, out, "Description=tensile e2e changed\n")
	assert.Contains(t, out, "NeedDaemonReload=no\n")
}
