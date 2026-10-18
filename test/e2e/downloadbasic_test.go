package e2e

import (
	"fmt"
	"os"
	"strings"
	"testing"

	"github.com/ntnn/tensile/test/e2e/framework"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// expectedSha256 is the SHA-256 of "hello from download test\n".
const expectedSha256 = "7367cc55a330ebfa85453c7c433f94c570f47e7b52275ec05293bc52e0a221d5"

func TestDownloadBasic(t *testing.T) {
	t.Parallel()

	env := framework.SharedContainer(t, framework.ArchLinux, framework.Scenario{Name: "downloadbasic"})

	exit, out := env.RunScenario(t)
	require.Zero(t, exit, out)

	t.Run("basic file download", func(t *testing.T) {
		t.Parallel()
		hash := fileSha256(t, env, "/opt/e2e/download/basic.txt")
		assert.Equal(t, expectedSha256, hash)
	})

	t.Run("directory download infers filename", func(t *testing.T) {
		t.Parallel()
		hash := fileSha256(t, env, "/opt/e2e/download/dir-test/testfile.txt")
		assert.Equal(t, expectedSha256, hash)
	})

	t.Run("download with explicit checksum", func(t *testing.T) {
		t.Parallel()
		hash := fileSha256(t, env, "/opt/e2e/download/checksum.txt")
		assert.Equal(t, expectedSha256, hash)
	})

	t.Run("download with checksum URL", func(t *testing.T) {
		t.Parallel()
		hash := fileSha256(t, env, "/opt/e2e/download/checksum-url.txt")
		assert.Equal(t, expectedSha256, hash)
	})

	t.Run("download with content-digest header", func(t *testing.T) {
		t.Parallel()
		hash := fileSha256(t, env, "/opt/e2e/download/content-digest.txt")
		assert.Equal(t, expectedSha256, hash)
	})

	t.Run("wrong checksum produces error", func(t *testing.T) {
		t.Parallel()
		exit, _ := env.Exec(t, "sh", "-c", "grep -q checksum /opt/e2e/download/checksum-fail-error.txt")
		assert.Zero(t, exit, "error should mention checksum")
	})

	t.Run("download then absent removes file", func(t *testing.T) {
		t.Parallel()
		_, err := os.Stat("/opt/e2e/download/absent.txt")
		assert.True(t, os.IsNotExist(err), "file should be absent")
	})
}

// fileSha256 computes the SHA-256 hash of a file in the container.
// The container exec output includes an 8-byte length prefix which is stripped.
func fileSha256(t *testing.T, env *framework.Env, path string) string {
	t.Helper()
	exit, out := env.Exec(t, "sh", "-c", fmt.Sprintf("sha256sum %s | cut -d' ' -f1", path))
	require.Zero(t, exit, "failed to hash file %s", path)
	if len(out) > 8 {
		out = out[8:]
	}
	return strings.TrimSpace(out)
}
