package e2e

import (
	"testing"

	"github.com/ntnn/tensile/test/e2e/framework"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestHandlerBasic(t *testing.T) {
	t.Parallel()

	env := framework.SharedContainer(t, framework.ArchLinux, "handlerbasic")

	exit, out := env.RunScenario(t)
	require.Zero(t, exit, out)

	exit, out = env.Exec(t, "test", "-f", "/opt/e2e/notified")
	assert.Zero(t, exit, "handler with executed notifier must run: %s", out)

	exit, out = env.Exec(t, "test", "-f", "/opt/e2e/silent")
	assert.NotZero(t, exit, "handler without notifiers must not run: %s", out)

	// second run: file unchanged, notifier skipped, handler skipped
	exit, out = env.Exec(t, "rm", "/opt/e2e/notified")
	require.Zero(t, exit, out)

	exit, out = env.RunScenario(t)
	require.Zero(t, exit, out)

	exit, out = env.Exec(t, "test", "-f", "/opt/e2e/notified")
	assert.NotZero(t, exit, "handler with unexecuted notifier must not run: %s", out)
}
