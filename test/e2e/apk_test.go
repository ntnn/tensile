package e2e

import (
	"testing"

	"github.com/ntnn/tensile/test/e2e/framework"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestApk(t *testing.T) {
	t.Parallel()

	env := framework.SharedContainer(t, framework.Alpine, framework.Scenario{Name: "apk"})

	// removal target for the scenario
	exit, out := env.Exec(t, "apk", "add", "less")
	require.Zero(t, exit, out)

	// pre-state: busybox and less installed, tree and jq not
	exit, out = env.Exec(t, "apk", "info", "-e", "busybox")
	require.Zero(t, exit, "busybox must be preinstalled: %s", out)

	exit, out = env.Exec(t, "apk", "info", "-e", "less")
	require.Zero(t, exit, "less must be installed before the run: %s", out)

	exit, out = env.Exec(t, "apk", "info", "-e", "tree")
	require.NotZero(t, exit, "tree must not be installed before the run: %s", out)

	exit, out = env.Exec(t, "apk", "info", "-e", "jq")
	require.NotZero(t, exit, "jq must not be installed before the run: %s", out)

	exit, out = env.RunScenario(t)
	require.Zero(t, exit, out)

	exit, out = env.Exec(t, "apk", "info", "-e", "busybox")
	assert.Zero(t, exit, "busybox must stay installed: %s", out)

	exit, out = env.Exec(t, "apk", "info", "-e", "tree")
	assert.Zero(t, exit, "tree must be installed: %s", out)

	exit, out = env.Exec(t, "apk", "info", "-e", "less")
	assert.NotZero(t, exit, "less must be removed: %s", out)

	exit, out = env.Exec(t, "apk", "info", "-e", "jq")
	assert.NotZero(t, exit, "jq must stay absent: %s", out)
}
