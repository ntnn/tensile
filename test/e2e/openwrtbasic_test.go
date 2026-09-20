package e2e

import (
	"testing"

	"github.com/ntnn/tensile/test/e2e/framework"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestOpenWrtBasic(t *testing.T) {
	t.Parallel()

	images := map[string]framework.Image{
		"24.10": framework.OpenWrt2410,
		"25.12": framework.OpenWrt2512,
	}

	for name, image := range images {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			env := framework.SharedContainer(t, image, framework.Scenario{Name: "openwrtbasic"})

			exit, out := env.RunScenario(t)
			require.Zero(t, exit, out)

			exit, out = env.Exec(t, "cat", "/opt/e2e/hello.txt")
			assert.Zero(t, exit, out)
			assert.Contains(t, out, "hello from tensile")

			exit, out = env.Exec(t, "test", "-f", "/opt/e2e/command-ran")
			assert.Zero(t, exit, "command must have run: %s", out)

			// rerun must be a no-op
			exit, out = env.RunScenario(t)
			assert.Zero(t, exit, out)
		})
	}
}
