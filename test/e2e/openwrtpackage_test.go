package e2e

import (
	"testing"

	"github.com/ntnn/tensile/test/e2e/framework"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestOpenWrtPackage(t *testing.T) {
	t.Parallel()

	for name, image := range framework.OpenWrtImages {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			env := framework.SharedContainer(t, image, framework.Scenario{Name: "openwrtpackage"})

			exit, out := env.Exec(t, "which", "tree")
			require.NotZero(t, exit, "tree must not be installed before the run: %s", out)

			exit, out = env.Exec(t, "which", "jq")
			require.NotZero(t, exit, "jq must not be installed before the run: %s", out)

			exit, out = env.RunScenario(t)
			require.Zero(t, exit, out)

			exit, out = env.Exec(t, "which", "tree")
			assert.Zero(t, exit, "tree must be installed: %s", out)

			exit, out = env.Exec(t, "which", "jq")
			assert.NotZero(t, exit, "jq must stay absent: %s", out)

			// rerun must be a no-op
			exit, out = env.RunScenario(t)
			assert.Zero(t, exit, out)
		})
	}
}
