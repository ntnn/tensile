package std

import (
	"fmt"
	"os"
	"runtime"
	"strings"

	"github.com/ntnn/tensile"
)

var _ tensile.Identifier = (*Facts)(nil)
var _ tensile.Executor = (*Facts)(nil)
var _ tensile.Reporter = (*Facts)(nil)

// FactsIdentity returns the identity of the facts node.
func FactsIdentity() tensile.Identity {
	return tensile.AsIdentity("facts")
}

// FactsData is the output reported by [Facts].
type FactsData struct {
	// OS is the operating system, e.g. "linux", "darwin".
	OS string
	// Architecture is the CPU architecture, e.g. "amd64", "arm64".
	Architecture string
	Hostname     string
	// Distribution is the ID from /etc/os-release, empty when unavailable.
	Distribution string
	// UserID is the uid of the running process.
	UserID int
}

// Facts gathers system facts and reports them as [FactsData].
// It changes nothing and reports on every run, including noop runs.
type Facts struct{}

// Identity implements [tensile.Identifier].
func (f *Facts) Identity() tensile.Identity {
	return FactsIdentity()
}

// NeedsExecution implements [tensile.Executor].
// Facts never execute, gathering happens in Report.
func (f *Facts) NeedsExecution(_ tensile.Wire) (bool, tensile.Diff, error) {
	return false, nil, nil
}

// Execute implements [tensile.Executor].
func (f *Facts) Execute(_ tensile.Wire) (tensile.Diff, error) {
	return nil, nil //nolint:nilnil // nil Diff is valid
}

// Report implements [tensile.Reporter].
func (f *Facts) Report(_ tensile.Wire) (any, error) {
	hostname, err := os.Hostname()
	if err != nil {
		return nil, fmt.Errorf("reading hostname: %w", err)
	}

	distribution, err := distribution()
	if err != nil {
		return nil, err
	}

	return FactsData{
		OS:           runtime.GOOS,
		Architecture: runtime.GOARCH,
		Hostname:     hostname,
		Distribution: distribution,
		UserID:       os.Getuid(),
	}, nil
}

// osReleasePath identifies the distribution on linux.
const osReleasePath = "/etc/os-release"

// distribution returns the ID from /etc/os-release.
// A missing file is not an error, the distribution is then empty.
func distribution() (string, error) {
	content, err := os.ReadFile(osReleasePath)
	if os.IsNotExist(err) {
		return "", nil
	}
	if err != nil {
		return "", fmt.Errorf("reading %s: %w", osReleasePath, err)
	}
	return parseOSReleaseID(string(content)), nil
}

// parseOSReleaseID returns the unquoted ID value from os-release content.
func parseOSReleaseID(content string) string {
	for line := range strings.Lines(content) {
		line = strings.TrimSpace(line)
		value, ok := strings.CutPrefix(line, "ID=")
		if !ok {
			continue
		}
		return strings.Trim(value, `"'`)
	}
	return ""
}
