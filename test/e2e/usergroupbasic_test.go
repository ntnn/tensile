package e2e

import (
	"testing"

	"github.com/ntnn/tensile/test/e2e/framework"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestUserGroupBasic(t *testing.T) {
	t.Parallel()

	env := framework.PrivateContainer(t, framework.ArchLinux, framework.Scenario{Name: "usergroupbasic"})

	// run twice: second run must be idempotent
	exit, out := env.RunScenario(t)
	require.Zero(t, exit, "first run: %s", out)

	exit, out = env.RunScenario(t)
	require.Zero(t, exit, "second run (idempotent): %s", out)

	// verify user exists with correct fields
	exit, out = env.Exec(t, "getent", "passwd", "e2e-alice")
	assert.Zero(t, exit, "user must exist: %s", out)
	assert.Contains(t, out, "e2e-alice")
	assert.Contains(t, out, ":1050:")
	assert.Contains(t, out, "/home/e2e-alice")
	assert.Contains(t, out, "/bin/bash")

	// verify groups: primary e2e-users, supplementary e2e-wheel
	exit, out = env.Exec(t, "id", "e2e-alice")
	assert.Zero(t, exit, "id must succeed: %s", out)
	assert.Contains(t, out, "e2e-alice")
	assert.Contains(t, out, "groups=")
	assert.Contains(t, out, "e2e-users")
	assert.Contains(t, out, "e2e-wheel")

	// verify group exists with correct GID
	exit, out = env.Exec(t, "getent", "group", "e2e-users")
	assert.Zero(t, exit, "e2e-users group must exist: %s", out)
	assert.Contains(t, out, "e2e-users:x:2050:")

	// verify wheel group exists
	exit, out = env.Exec(t, "getent", "group", "e2e-wheel")
	assert.Zero(t, exit, "e2e-wheel group must exist: %s", out)
}
