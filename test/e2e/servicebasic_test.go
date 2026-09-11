package e2e

import (
	"testing"

	"github.com/ntnn/tensile/test/e2e/framework"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestServiceBasic(t *testing.T) {
	t.Parallel()

	env := framework.Start(t, framework.SystemdDebian, "servicebasic")

	exit, out := env.RunScenario(t)
	require.Zero(t, exit, out)

	// verification bypasses tensile on purpose
	exit, out = env.Exec(t, "cat", "/opt/e2e/hello.txt")
	assert.Zero(t, exit, out)
	assert.Contains(t, out, "hello from tensile")

	exit, out = env.Exec(t, "readlink", "/opt/e2e/link")
	assert.Zero(t, exit, out)
	assert.Contains(t, out, "/opt/e2e/hello.txt")

	exit, out = env.Exec(t, "systemctl", "is-active", "e2e-dummy.service")
	assert.Zero(t, exit, out)

	exit, out = env.Exec(t, "systemctl", "is-enabled", "e2e-dummy.service")
	assert.Zero(t, exit, out)

	exit, out = env.RunScenario(t, "-verify-noop")
	assert.Zero(t, exit, "second run must not execute any node: %s", out)
}
