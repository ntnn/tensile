package e2e

import (
	"testing"

	"github.com/ntnn/tensile/test/e2e/framework"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestOpenWrtBasic(t *testing.T) {
	t.Parallel()

	cases := map[string]struct {
		image framework.Image
		// update refreshes the package lists, required by manager
		// detection and install.
		update []string
	}{
		"24.10": {image: framework.OpenWrt2410, update: []string{"opkg", "update"}},
		"25.12": {image: framework.OpenWrt2512, update: []string{"apk", "update"}},
	}

	for name, cas := range cases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			env := framework.SharedContainer(t, cas.image, framework.Scenario{Name: "openwrtbasic"})

			exit, out := env.Exec(t, cas.update...)
			require.Zero(t, exit, out)

			// pre-state: tree not installed, cron disabled and stopped
			exit, out = env.Exec(t, "which", "tree")
			require.NotZero(t, exit, "tree must not be installed before the run: %s", out)

			exit, out = env.Exec(t, "/etc/init.d/cron", "running")
			require.NotZero(t, exit, "cron must not run before the run: %s", out)

			exit, out = env.RunScenario(t)
			require.Zero(t, exit, out)

			exit, out = env.Exec(t, "cat", "/opt/e2e/hello.txt")
			assert.Zero(t, exit, out)
			assert.Contains(t, out, "hello from tensile")

			exit, out = env.Exec(t, "test", "-f", "/opt/e2e/command-ran")
			assert.Zero(t, exit, "command must have run: %s", out)

			exit, out = env.Exec(t, "which", "tree")
			assert.Zero(t, exit, "tree must be installed: %s", out)

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
