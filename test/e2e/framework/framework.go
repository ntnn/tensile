package framework

import (
	"context"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"sync"
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

// sharedContainers caches running containers by image ref.
var (
	sharedLock       sync.Mutex
	sharedContainers = map[string]testcontainers.Container{}
)

// SharedContainer deploys the scenario ./<name>/ into a container shared by all tests requesting the same image.
// Shared containers are reaped by ryuk after the test process exits.
//
// Shared containers should be preferred for testing, since the startup
// still takes time, CPU and RAM and in general tests should be able to
// be written in a way to not impact each other.
// For cases where a test will impact other tests use [PrivateContainer].
func SharedContainer(t *testing.T, image Image, name string) *Env {
	t.Helper()

	ctr, err := sharedContainer(image)
	require.NoError(t, err, "starting shared container %q", image.Ref)

	return deployScenario(t, ctr, name)
}

// PrivateContainer deploys the scenario ./<name>/ into a dedicated container that is terminated when the test exits.
func PrivateContainer(t *testing.T, image Image, name string) *Env {
	t.Helper()

	ctr, err := startContainer(t.Context(), image)
	testcontainers.CleanupContainer(t, ctr)
	require.NoError(t, err, "starting container %q", image.Ref)

	return deployScenario(t, ctr, name)
}

// sharedContainer returns the shared container for image.
// If no cntainer for image exists it is started.
func sharedContainer(image Image) (testcontainers.Container, error) {
	sharedLock.Lock()
	defer sharedLock.Unlock()

	if ctr, ok := sharedContainers[image.Ref]; ok {
		return ctr, nil
	}

	// context.Background: the container outlives the requesting test
	ctr, err := startContainer(context.Background(), image)
	if err != nil {
		return nil, err
	}
	sharedContainers[image.Ref] = ctr
	return ctr, nil
}

func startContainer(ctx context.Context, image Image) (testcontainers.Container, error) {
	return testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		Started:    true,
		Image:      image.Ref,
		Entrypoint: image.Entrypoint,
		WaitingFor: wait.ForExec(image.WaitCmd).WithExitCodeMatcher(ready),
		HostConfigModifier: func(hc *container.HostConfig) {
			hc.CgroupnsMode = container.CgroupnsModePrivate
			hc.CapAdd = image.CapAdd
			hc.SecurityOpt = image.SecurityOpt
			hc.Tmpfs = image.Tmpfs
		},
	})
}

// deployScenario builds ./<name>/ and copies it to /usr/local/bin/<name>.
func deployScenario(t *testing.T, ctr testcontainers.Container, name string) *Env {
	t.Helper()

	bin := buildScenario(t, name)
	binPath := "/usr/local/bin/" + name

	err := ctr.CopyFileToContainer(t.Context(), bin, binPath, containerFileMode)
	require.NoError(t, err, "deploying scenario %q", name)

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
