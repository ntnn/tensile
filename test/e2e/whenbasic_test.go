package e2e

import (
	"crypto/sha256"
	"fmt"
	"testing"

	"github.com/ntnn/tensile/test/e2e/framework"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestWhenBasic(t *testing.T) {
	t.Parallel()

	env := framework.SharedContainer(t, framework.ArchLinux, "whenbasic")

	exit, out := env.RunScenario(t)
	require.Zero(t, exit, out)

	exit, out = env.Exec(t, "cat", "/opt/e2e/when-true")
	require.Zero(t, exit, out)
	assert.Contains(t, out, "enabled", "enabled node must execute")

	wantHash := fmt.Sprintf("%x", sha256.Sum256([]byte("disabled\n")))
	exit, out = env.Exec(t, "cat", "/opt/e2e/when-false-report")
	require.Zero(t, exit, out)
	assert.Contains(t, out, wantHash, "disabled node must still report for dependers")
}
