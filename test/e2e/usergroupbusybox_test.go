package e2e

import (
	"testing"

	"github.com/ntnn/tensile/test/e2e/framework"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestUserGroupBusybox(t *testing.T) {
	t.Parallel()

	env := framework.PrivateContainer(t, framework.Busybox, framework.Scenario{Name: "usergroupbusybox"})

	exit, out := env.RunScenario(t)
	require.Zero(t, exit, "scenario must succeed: %s", out)

	// busybox doesn't have getent, use grep on passwd and group
	exit, out = env.Exec(t, "grep", "^e2e-busybox-user:", "/etc/passwd")
	assert.Zero(t, exit, "user must exist: %s", out)
	assert.Contains(t, out, "e2e-busybox-user:x:1080:2080")

	exit, out = env.Exec(t, "grep", "^e2e-busybox-group:", "/etc/group")
	assert.Zero(t, exit, "group must exist: %s", out)
	assert.Contains(t, out, "e2e-busybox-group:x:2080:")

	// membership: user listed in group entry
	exit, out = env.Exec(t, "grep", "^e2e-busybox-group:", "/etc/group")
	assert.Zero(t, exit, "group: %s", out)
	assert.Contains(t, out, "e2e-busybox-user", "user must be member: %s", out)
}
