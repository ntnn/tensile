package framework

import (
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go"
)

// Scenario builds and deploys a test binary into a container.
type Scenario struct {
	// Name is the scenario directory ./<Name>/ and the binary name.
	Name string
	// Compiler is the compiler binary.
	// Empty means "go".
	Compiler string
	// Args are the compiler arguments before the output and package arguments.
	// Empty means ["build"].
	Args []string
	// Env is extra build environment appended after the defaults.
	Env []string
}

// Build compiles ./<Name>/ as a static binary returns the binary path.
func (s Scenario) Build(t *testing.T) string {
	t.Helper()

	compiler := s.Compiler
	if compiler == "" {
		compiler = "go"
	}
	args := s.Args
	if len(args) == 0 {
		args = []string{"build"}
	}

	out := filepath.Join(t.TempDir(), s.Name)
	args = append(args, "-o", out, "./"+s.Name)

	//nolint:gosec // passing variables is expected
	cmd := exec.CommandContext(t.Context(), compiler, args...)
	cmd.Env = append(os.Environ(),
		"CGO_ENABLED=0",
		"GOOS=linux",
		"GOARCH="+runtime.GOARCH,
	)
	cmd.Env = append(cmd.Env, s.Env...)
	output, err := cmd.CombinedOutput()
	require.NoError(t, err, "building scenario %q: %s", s.Name, output)
	return out
}

// Deploy builds the scenario and copies it to /usr/local/bin/tensile-<Name>.
func (s Scenario) Deploy(t *testing.T, ctr testcontainers.Container) *Env {
	t.Helper()

	bin := s.Build(t)
	binPath := "/usr/local/bin/tensile-" + s.Name

	err := ctr.CopyFileToContainer(t.Context(), bin, binPath, containerFileMode)
	require.NoError(t, err, "deploying scenario %q", s.Name)

	return &Env{
		container: ctr,
		binPath:   binPath,
	}
}
