package e2e

import (
	"testing"

	"github.com/ntnn/tensile/test/e2e/framework"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Bug: without package lists no manager handles a package that is not installed,
// so noop runs on a fresh system failed detection.
func TestOpenWrtPackageNoop(t *testing.T) {
	t.Parallel()

	for name, image := range framework.OpenWrtImages {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			// shared containers may already have package lists
			env := framework.PrivateContainer(t, image, framework.Scenario{Name: "openwrtpackagenoop"})

			exit, out := env.Exec(t, "which", "tree")
			require.NotZero(t, exit, "tree must not be installed before the run: %s", out)

			exit, out = env.RunScenario(t, "-noop")
			require.Zero(t, exit, out)
			assert.Contains(t, out, "packages\n  tree: absent -> present", "install must be planned")
			assert.NotContains(t, out, "jq:", "absent must be satisfied")

			exit, out = env.Exec(t, "which", "tree")
			assert.NotZero(t, exit, "noop must not install tree: %s", out)

			// unknown after updating lists is a user error
			exit, out = env.RunScenario(t, "-unknown")
			assert.NotZero(t, exit, out)
			assert.Contains(t, out, "no registered package manager handles \"package-does-not-exist\"")
		})
	}
}
