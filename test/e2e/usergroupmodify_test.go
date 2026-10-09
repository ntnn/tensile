package e2e

import (
	"testing"

	"github.com/ntnn/tensile/test/e2e/framework"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

//nolint:tparallel // subtests share container state and must run sequentially
func TestUserGroupModify(t *testing.T) {
	t.Parallel()

	env := framework.PrivateContainer(t, framework.ArchLinux, framework.Scenario{Name: "usergroupmodify"})

	// phase 1: create managed entities
	//nolint:paralleltest // sequential subtest sharing container state
	t.Run("phase1-create", func(t *testing.T) {
		exit, out := env.RunScenario(t, "create")
		require.Zero(t, exit, "phase 1: %s", out)

		exit, out = env.Exec(t, "getent", "passwd", "e2e-managed1")
		assert.Zero(t, exit, "managed1 must exist: %s", out)
		assert.Contains(t, out, ":1060:")

		exit, out = env.Exec(t, "getent", "passwd", "e2e-managed2")
		assert.Zero(t, exit, "managed2 must exist: %s", out)
		assert.Contains(t, out, ":1061:")

		exit, out = env.Exec(t, "getent", "group", "e2e-team")
		assert.Zero(t, exit, "e2e-team must exist: %s", out)
		assert.Contains(t, out, "e2e-team:x:2060:")

		exit, out = env.Exec(t, "id", "e2e-managed1")
		assert.Zero(t, exit, "id managed1: %s", out)
		assert.Contains(t, out, "e2e-team")

		exit, out = env.Exec(t, "id", "e2e-managed2")
		assert.Zero(t, exit, "id managed2: %s", out)
		assert.Contains(t, out, "e2e-team")
	})

	// create unmanaged entities for pruning
	//nolint:paralleltest // sequential subtest sharing container state
	t.Run("setup-unmanaged", func(t *testing.T) {
		exit, out := env.Exec(t, "useradd", "-u", "1062", "e2e-orphan")
		require.Zero(t, exit, "create orphan user: %s", out)

		exit, out = env.Exec(t, "groupadd", "-g", "2062", "e2e-orphan-group")
		require.Zero(t, exit, "create orphan group: %s", out)

		exit, out = env.Exec(t, "usermod", "-aG", "e2e-team", "e2e-orphan")
		require.Zero(t, exit, "add orphan to e2e-team: %s", out)
	})

	// phase 2: prune unmanaged
	//nolint:paralleltest // sequential subtest sharing container state
	t.Run("phase2-prune", func(t *testing.T) {
		exit, out := env.RunScenario(t, "prune")
		require.Zero(t, exit, "phase 2: %s", out)

		// managed entities survive
		exit, out = env.Exec(t, "getent", "passwd", "e2e-managed1")
		assert.Zero(t, exit, "managed1 survives pruning: %s", out)

		exit, out = env.Exec(t, "getent", "passwd", "e2e-managed2")
		assert.Zero(t, exit, "managed2 survives pruning: %s", out)

		exit, out = env.Exec(t, "getent", "group", "e2e-team")
		assert.Zero(t, exit, "e2e-team survives pruning: %s", out)

		// unmanaged orphan user deleted
		exit, out = env.Exec(t, "getent", "passwd", "e2e-orphan")
		assert.NotZero(t, exit, "orphan user must be pruned: %s", out)

		// unmanaged orphan group deleted
		exit, out = env.Exec(t, "getent", "group", "e2e-orphan-group")
		assert.NotZero(t, exit, "orphan group must be pruned: %s", out)

		// orphan's membership in e2e-team removed
		exit, out = env.Exec(t, "getent", "group", "e2e-team")
		assert.Zero(t, exit, "e2e-team must still exist: %s", out)
		assert.NotContains(t, out, "e2e-orphan", "orphan must not be in e2e-team: %s", out)
	})

	// phase 3: modify user shell and group GID
	//nolint:paralleltest // sequential subtest sharing container state
	t.Run("phase3-modify", func(t *testing.T) {
		exit, out := env.RunScenario(t, "modify")
		require.Zero(t, exit, "phase 3: %s", out)

		exit, out = env.Exec(t, "getent", "passwd", "e2e-managed1")
		assert.Zero(t, exit, "managed1: %s", out)
		assert.Contains(t, out, "/bin/sh", "shell changed to /bin/sh: %s", out)

		exit, out = env.Exec(t, "getent", "group", "e2e-team")
		assert.Zero(t, exit, "e2e-team: %s", out)
		assert.Contains(t, out, "e2e-team:x:2061:", "GID changed to 2061: %s", out)
	})

	// phase 4: remove managed2 declaration, prune
	//nolint:paralleltest // sequential subtest sharing container state
	t.Run("phase4-remove", func(t *testing.T) {
		exit, out := env.RunScenario(t, "remove")
		require.Zero(t, exit, "phase 4: %s", out)

		// managed1 survives
		exit, out = env.Exec(t, "getent", "passwd", "e2e-managed1")
		assert.Zero(t, exit, "managed1 survives: %s", out)

		// managed2 removed
		exit, out = env.Exec(t, "getent", "passwd", "e2e-managed2")
		assert.NotZero(t, exit, "managed2 must be pruned: %s", out)

		// e2e-team survives (still claimed by managed1)
		exit, out = env.Exec(t, "getent", "group", "e2e-team")
		assert.Zero(t, exit, "e2e-team survives: %s", out)

		// managed2's membership removed, managed1 still in
		assert.NotContains(t, out, "e2e-managed2", "managed2 not in e2e-team: %s", out)
		assert.Contains(t, out, "e2e-managed1", "managed1 still in e2e-team: %s", out)
	})

	// phase 5: empty groups removes all supplementary
	//nolint:paralleltest // sequential subtest sharing container state
	t.Run("phase5-empty-groups", func(t *testing.T) {
		exit, out := env.RunScenario(t, "empty-groups")
		require.Zero(t, exit, "phase 5: %s", out)

		exit, out = env.Exec(t, "id", "e2e-managed1")
		assert.Zero(t, exit, "id managed1: %s", out)
		assert.NotContains(t, out, "e2e-team", "empty groups removed e2e-team: %s", out)
	})

	// phase 6: conflict detection
	//nolint:paralleltest // sequential subtest sharing container state
	t.Run("phase6-conflict", func(t *testing.T) {
		exit, out := env.RunScenario(t, "conflict")
		require.Zero(t, exit, "conflict phase must exit 0: %s", out)
		assert.Contains(t, out, "conflict detected", "must detect conflict: %s", out)
		assert.Contains(t, out, "already claimed", "must mention already claimed: %s", out)
	})

	// phase 7: prune unmanaged wheel membership, keep managed
	//nolint:paralleltest // sequential subtest sharing container state
	t.Run("phase7-prune-wheel", func(t *testing.T) {
		// setup: create unmanaged wheel member
		exit, out := env.Exec(t, "useradd", "-u", "1076", "e2e-wheel-unmanaged")
		require.Zero(t, exit, "create unmanaged wheel user: %s", out)
		exit, out = env.Exec(t, "usermod", "-aG", "wheel", "e2e-wheel-unmanaged")
		require.Zero(t, exit, "add unmanaged to wheel: %s", out)

		exit, out = env.RunScenario(t, "prune-wheel")
		require.Zero(t, exit, "prune wheel phase: %s", out)

		// managed wheel membership kept
		exit, out = env.Exec(t, "id", "e2e-wheel-managed")
		assert.Zero(t, exit, "id wheel managed: %s", out)
		assert.Contains(t, out, "groups=", "id managed: %s", out)
		assert.Contains(t, out, "wheel", "managed wheel membership kept: %s", out)

		// unmanaged wheel membership pruned
		exit, out = env.Exec(t, "id", "e2e-wheel-unmanaged")
		assert.Zero(t, exit, "id wheel unmanaged: %s", out)
		assert.NotContains(t, out, "998", "unmanaged wheel membership pruned: %s", out)
	})
}
