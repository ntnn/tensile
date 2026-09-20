package e2e

import (
	"testing"

	"github.com/ntnn/tensile/test/e2e/framework"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestOpenWrtService(t *testing.T) {
	t.Parallel()

	for name, image := range framework.OpenWrtImages {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			env := framework.SharedContainer(t, image, framework.Scenario{Name: "openwrtservice"})

			// pre-state: cron stopped
			exit, out := env.Exec(t, "/etc/init.d/cron", "running")
			require.NotZero(t, exit, "cron must not run before the run: %s", out)

			exit, out = env.RunScenario(t)
			require.Zero(t, exit, out)

			exit, out = env.Exec(t, "/etc/init.d/cron", "enabled")
			assert.Zero(t, exit, "cron must be enabled: %s", out)

			exit, out = env.Exec(t, "/etc/init.d/cron", "running")
			assert.Zero(t, exit, "cron must be running: %s", out)

			// rerun must be a no-op
			exit, out = env.RunScenario(t)
			assert.Zero(t, exit, out)
		})
	}
}
