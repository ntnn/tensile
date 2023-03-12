package facts

import (
	"fmt"
	"os"
	"runtime"
)

type Facts struct {
	Env map[string]string `json:"env"`

	// GOARCH from runtime
	GOARCH string `json:"goarch"`
	// GOOS from runtime
	GOOS string `json:"goos"`

	OSRelease OSRelease `json:"os_release"`

	// Executable is the base name of the running binary.
	// ExecutablePath is the fully qualified path of the running binary.
	Executable     string `json:"executable"`
	ExecutablePath string `json:"executable_path"`

	Workdir  string `json:"workdir"`
	Hostname string `json:"hostname"`

	Custom map[string]any `json:"custom"`
}

func New() (Facts, error) {
	f := Facts{}

	f.GOARCH = runtime.GOARCH
	f.GOOS = runtime.GOOS

	rel, err := NewOSRelease()
	if err != nil {
		return Facts{}, fmt.Errorf("error getting OSRelease: %w", err)
	}
	f.OSRelease = rel

	f.Env = Env()

	if err := f.executable(); err != nil {
		return Facts{}, err
	}

	if err := f.workdir(); err != nil {
		return Facts{}, err
	}

	if err := f.hostname(); err != nil {
		return Facts{}, err
	}

	return f, nil
}

func (f *Facts) workdir() error {
	s, err := os.Getwd()
	if err != nil {
		return err
	}
	f.Workdir = s
	return nil
}

func (f *Facts) hostname() error {
	s, err := os.Hostname()
	if err != nil {
		return err
	}
	f.Hostname = s
	return nil
}
