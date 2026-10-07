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
	// OSRelease is parsed from /etc/os-release, zero when unavailable.
	OSRelease OSRelease
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

	osRelease, err := readOSRelease()
	if err != nil {
		return nil, err
	}

	return FactsData{
		OS:           runtime.GOOS,
		Architecture: runtime.GOARCH,
		Hostname:     hostname,
		OSRelease:    osRelease,
		UserID:       os.Getuid(),
	}, nil
}

// osReleasePath identifies the distribution on linux.
const osReleasePath = "/etc/os-release"

// OSRelease holds the identifying fields of os-release(5).
type OSRelease struct {
	// Name is NAME, e.g. "Arch Linux".
	Name string
	// PrettyName is PRETTY_NAME, e.g. "Arch Linux".
	PrettyName string
	// ID is ID, e.g. "arch".
	ID string
	// IDLike is the space separated ID_LIKE, e.g. ["rhel", "fedora"].
	IDLike []string
	// Version is VERSION, e.g. "9.4 (Blue Onyx)".
	Version string
	// VersionID is VERSION_ID, e.g. "9.4".
	VersionID string
	// BuildID is BUILD_ID, e.g. "rolling".
	BuildID string
}

// readOSRelease parses /etc/os-release.
// A missing file is not an error, the result is then zero.
func readOSRelease() (OSRelease, error) {
	content, err := os.ReadFile(osReleasePath)
	if os.IsNotExist(err) {
		return OSRelease{}, nil
	}
	if err != nil {
		return OSRelease{}, fmt.Errorf("reading %s: %w", osReleasePath, err)
	}
	return parseOSRelease(string(content)), nil
}

// parseOSRelease returns the known fields from os-release content.
func parseOSRelease(content string) OSRelease {
	values := parseOSReleaseValues(content)
	osRelease := OSRelease{
		Name:       values["NAME"],
		PrettyName: values["PRETTY_NAME"],
		ID:         values["ID"],
		Version:    values["VERSION"],
		VersionID:  values["VERSION_ID"],
		BuildID:    values["BUILD_ID"],
	}
	// strings.Fields returns an empty slice, keep nil when absent.
	if idLike := values["ID_LIKE"]; idLike != "" {
		osRelease.IDLike = strings.Fields(idLike)
	}
	return osRelease
}

// parseOSReleaseValues returns the unquoted values of os-release content by key.
func parseOSReleaseValues(content string) map[string]string {
	values := map[string]string{}
	for line := range strings.Lines(content) {
		key, value, ok := strings.Cut(strings.TrimSpace(line), "=")
		if !ok {
			continue
		}
		values[key] = strings.Trim(value, `"'`)
	}
	return values
}
