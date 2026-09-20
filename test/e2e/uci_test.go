package e2e

import (
	"testing"

	"github.com/ntnn/tensile/test/e2e/framework"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestUci(t *testing.T) {
	t.Parallel()

	for name, image := range framework.OpenWrtImages {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			env := framework.SharedContainer(t, image, framework.Scenario{Name: "uci"})

			// seed tensile: main section with a removal-target option,
			// an obsolete section, two anonymous host sections and a
			// named one to keep; tensile2: empty config file
			exit, out := env.Exec(t, "sh", "-c",
				"touch /etc/config/tensile /etc/config/tensile2"+
					" && uci set tensile.main=settings"+
					" && uci set tensile.main.legacy=old"+
					" && uci set tensile.obsolete=settings"+
					" && uci add tensile host"+
					" && uci add tensile host"+
					" && uci set tensile.keep=host"+
					" && uci commit")
			require.Zero(t, exit, out)

			exit, out = env.Exec(t, "sh", "-c", "uci show tensile | grep -c '@host'")
			require.Zero(t, exit, "anonymous host sections must be seeded: %s", out)

			exit, out = env.RunScenario(t)
			require.Zero(t, exit, out)

			// scalar option per value type
			exit, out = env.Exec(t, "uci", "get", "tensile.main.hello")
			assert.Zero(t, exit, out)
			assert.Contains(t, out, "world")

			exit, out = env.Exec(t, "uci", "get", "tensile.main.port")
			assert.Zero(t, exit, out)
			assert.Contains(t, out, "8080")

			exit, out = env.Exec(t, "uci", "get", "tensile.main.ignore")
			assert.Zero(t, exit, out)
			assert.Contains(t, out, "1", "bool must render as 1")

			// list options keep declared order
			exit, out = env.Exec(t, "uci", "show", "tensile.main.dns")
			assert.Zero(t, exit, out)
			assert.Contains(t, out, "'192.168.178.5' '1.1.1.1'", "dns list must keep declared order")

			exit, out = env.Exec(t, "uci", "show", "tensile.main.ports")
			assert.Zero(t, exit, out)
			assert.Contains(t, out, "'80' '443'", "int list must keep declared order")

			// absent option and section are removed
			exit, out = env.Exec(t, "uci", "get", "tensile.main.legacy")
			assert.NotZero(t, exit, "legacy option must be removed: %s", out)

			exit, out = env.Exec(t, "uci", "get", "tensile.obsolete")
			assert.NotZero(t, exit, "obsolete section must be removed: %s", out)

			// anonymous host sections wiped, named ones intact
			exit, out = env.Exec(t, "sh", "-c", "uci show tensile | grep '@host'")
			assert.NotZero(t, exit, "anonymous host sections must be gone: %s", out)

			exit, out = env.Exec(t, "uci", "get", "tensile.keep")
			assert.Zero(t, exit, "named host section must be kept: %s", out)
			assert.Contains(t, out, "host", "kept section must still be type host")

			exit, out = env.Exec(t, "uci", "get", "tensile.srv.name")
			assert.Zero(t, exit, "srv section and option must exist: %s", out)
			assert.Contains(t, out, "srv")

			// exactly the named host sections keep and srv remain
			exit, out = env.Exec(t, "sh", "-c", "uci show tensile | grep -c '=host'")
			assert.Zero(t, exit, out)
			assert.Contains(t, out, "2", "exactly keep and srv must remain: %s", out)

			// committed to the config files, not only staged
			exit, out = env.Exec(t, "grep", "-q", "option hello 'world'", "/etc/config/tensile")
			assert.Zero(t, exit, "hello must be committed: %s", out)

			exit, out = env.Exec(t, "grep", "-q", "list dns '192.168.178.5'", "/etc/config/tensile")
			assert.Zero(t, exit, "dns list must be committed: %s", out)

			// second config is committed only by the global handler
			exit, out = env.Exec(t, "grep", "-q", "option greet 'hi'", "/etc/config/tensile2")
			assert.Zero(t, exit, "greet must be committed by the global handler: %s", out)

			exit, out = env.Exec(t, "sh", "-c", "[ -z \"$(uci changes)\" ]")
			assert.Zero(t, exit, "nothing may stay staged after commit: %s", out)

			// rerun must be a no-op
			exit, out = env.RunScenario(t)
			assert.Zero(t, exit, out)
		})
	}
}
