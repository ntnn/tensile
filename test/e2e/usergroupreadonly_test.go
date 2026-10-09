package e2e

import (
	"testing"

	"github.com/ntnn/tensile/test/e2e/framework"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

//nolint:tparallel // subtests share container state and must run sequentially
func TestUserGroupReadOnly(t *testing.T) {
	t.Parallel()

	env := framework.PrivateContainer(t, framework.ArchLinux, framework.Scenario{Name: "usergroupreadonly"})

	// create entities
	//nolint:paralleltest // sequential subtest sharing container state
	t.Run("phase1-create", func(t *testing.T) {
		exit, out := env.RunScenario(t, "create")
		require.Zero(t, exit, "phase 1: %s", out)

		exit, out = env.Exec(t, "getent", "passwd", "e2e-readonly-user")
		assert.Zero(t, exit, "user must exist: %s", out)
		assert.Contains(t, out, ":1070:")
		assert.Contains(t, out, "/bin/bash")

		exit, out = env.Exec(t, "getent", "group", "e2e-readonly-group")
		assert.Zero(t, exit, "group must exist: %s", out)
		assert.Contains(t, out, "e2e-readonly-group:x:2070:")
	})

	// readonly with drift: no changes
	//nolint:paralleltest // sequential subtest sharing container state
	t.Run("phase2-readonly", func(t *testing.T) {
		exit, out := env.RunScenario(t, "readonly")
		require.Zero(t, exit, "phase 2: %s", out)

		// user unchanged: still uid 1070, shell /bin/bash
		exit, out = env.Exec(t, "getent", "passwd", "e2e-readonly-user")
		assert.Zero(t, exit, "user: %s", out)
		assert.Contains(t, out, ":1070:", "uid unchanged: %s", out)
		assert.Contains(t, out, "/bin/bash", "shell unchanged: %s", out)

		// group unchanged: still gid 2070
		exit, out = env.Exec(t, "getent", "group", "e2e-readonly-group")
		assert.Zero(t, exit, "group: %s", out)
		assert.Contains(t, out, "e2e-readonly-group:x:2070:", "gid unchanged: %s", out)
	})

	// readonly membership: rejected at validate
	//nolint:paralleltest // sequential subtest sharing container state
	t.Run("phase3-readonly-membership", func(t *testing.T) {
		exit, out := env.RunScenario(t, "readonly-membership")
		require.NotZero(t, exit, "phase 3 must fail validation: %s", out)
		assert.Contains(t, out, "unknown state", "must reject readonly membership: %s", out)
	})
}
