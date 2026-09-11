package e2e

import (
	"testing"

	"github.com/ntnn/tensile/test/e2e/framework"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPackagePacman(t *testing.T) {
	t.Parallel()

	env := framework.Start(t, framework.ArchLinux, "packagepacman")

	exit, out := env.Exec(t, "pacman", "-Sy", "--noconfirm")
	require.Zero(t, exit, out)

	// removal target for the scenario
	exit, out = env.Exec(t, "pacman", "-S", "--noconfirm", "--needed", "less")
	require.Zero(t, exit, out)

	// pre-state: gzip and less installed, tree not
	exit, out = env.Exec(t, "pacman", "-Q", "gzip")
	require.Zero(t, exit, "gzip must be preinstalled: %s", out)

	exit, out = env.Exec(t, "pacman", "-Q", "less")
	require.Zero(t, exit, "less must be installed before the run: %s", out)

	exit, out = env.Exec(t, "pacman", "-Q", "tree")
	require.NotZero(t, exit, "tree must not be installed before the run: %s", out)

	exit, out = env.Exec(t, "pacman", "-Q", "jq")
	require.NotZero(t, exit, "jq must not be installed before the run: %s", out)

	exit, out = env.RunScenario(t)
	require.Zero(t, exit, out)

	exit, out = env.Exec(t, "pacman", "-Q", "gzip")
	assert.Zero(t, exit, "gzip must stay installed: %s", out)

	exit, out = env.Exec(t, "pacman", "-Q", "tree")
	assert.Zero(t, exit, "tree must be installed: %s", out)

	exit, out = env.Exec(t, "pacman", "-Q", "less")
	assert.NotZero(t, exit, "less must be removed: %s", out)

	exit, out = env.Exec(t, "pacman", "-Q", "jq")
	assert.NotZero(t, exit, "jq must stay absent: %s", out)
}
