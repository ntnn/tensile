package e2e

import (
	"crypto/sha256"
	"fmt"
	"testing"

	"github.com/ntnn/tensile/test/e2e/framework"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestReporterBasic(t *testing.T) {
	t.Parallel()

	env := framework.SharedContainer(t, framework.ArchLinux, "reporterbasic")

	exit, out := env.RunScenario(t)
	require.Zero(t, exit, out)

	wantHash := fmt.Sprintf("%x", sha256.Sum256([]byte("hello from reporter\n")))

	exit, out = env.Exec(t, "cat", "/opt/e2e/reporter-hash")
	require.Zero(t, exit, out)
	assert.Contains(t, out, wantHash, "consumer must write the reported file hash")

	exit, out = env.Exec(t, "cat", "/opt/e2e/reporter-cmd")
	require.Zero(t, exit, out)
	assert.Contains(t, out, "hello reporter", "consumer must write the reported command output")

	// second run: file unchanged, FileContent skips execution but must
	// still report so the consumer can read the output again
	exit, out = env.Exec(t, "rm", "/opt/e2e/reporter-hash")
	require.Zero(t, exit, out)

	exit, out = env.RunScenario(t)
	require.Zero(t, exit, out)

	exit, out = env.Exec(t, "cat", "/opt/e2e/reporter-hash")
	require.Zero(t, exit, out)
	assert.Contains(t, out, wantHash, "unchanged node must still report its output")
}
