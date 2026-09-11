package framework

import (
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/moby/moby/api/types/container"
	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"
)

// Env is a running container machine with a deployed scenario binary.
type Env struct {
	container testcontainers.Container
	binPath   string
}

const containerFileMode = 0o755

// Start builds the scenario ./<name>/, starts the image and deploys the binary to /usr/local/bin/<name>.
func Start(t *testing.T, image Image, name string) *Env {
	t.Helper()

	bin := buildScenario(t, name)
	binPath := "/usr/local/bin/" + name

	ctr, err := testcontainers.GenericContainer(t.Context(), testcontainers.GenericContainerRequest{
		Started:    true,
		Image:      image.Ref,
		Entrypoint: image.Entrypoint,
		WaitingFor: wait.ForExec(image.WaitCmd).WithExitCodeMatcher(ready),
		HostConfigModifier: func(hc *container.HostConfig) {
			hc.CgroupnsMode = container.CgroupnsModePrivate
			hc.CapAdd = image.CapAdd
			hc.Tmpfs = image.Tmpfs
		},
		Files: []testcontainers.ContainerFile{
			{
				HostFilePath:      bin,
				ContainerFilePath: binPath,
				FileMode:          containerFileMode,
			},
		},
	})
	testcontainers.CleanupContainer(t, ctr)
	require.NoError(t, err, "starting container %q", image.Ref)

	return &Env{
		container: ctr,
		binPath:   binPath,
	}
}

// Exec executes a command in the container machine.
func (env *Env) Exec(t *testing.T, cmd ...string) (int, string) {
	t.Helper()

	exit, reader, err := env.container.Exec(t.Context(), cmd)
	require.NoError(t, err, "executing %v in container", cmd)
	out, err := io.ReadAll(reader)
	require.NoError(t, err, "reading output of %v", cmd)
	return exit, string(out)
}

// RunScenario executes the deployed scenario binary with args.
func (env *Env) RunScenario(t *testing.T, args ...string) (int, string) {
	t.Helper()

	return env.Exec(t, append([]string{env.binPath}, args...)...)
}

// buildScenario compiles ./<name>/ as a static linux binary for the
// container platform and returns the binary path.
func buildScenario(t *testing.T, name string) string {
	t.Helper()

	out := filepath.Join(t.TempDir(), name)
	//nolint:gosec // passing variables is expected
	cmd := exec.CommandContext(t.Context(), "go", "build", "-o", out, "./"+name)
	cmd.Env = append(os.Environ(),
		"CGO_ENABLED=0",
		"GOOS=linux",
		"GOARCH="+runtime.GOARCH,
	)
	output, err := cmd.CombinedOutput()
	require.NoError(t, err, "building scenario %q: %s", name, output)

	return out
}

// ready accepts a completed boot even when systemd reports degraded.
func ready(exitCode int) bool {
	return exitCode == 0 || exitCode == 1
}
