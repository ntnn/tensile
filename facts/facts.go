package facts

import "os"

type ctxFacts string

const CtxFacts ctxFacts = "ctxFacts"

type Facts struct {
	Env map[string]string

	// Executable is the base name of the running binary.
	// ExecutablePath is the fully qualified path of the running binary.
	Executable, ExecutablePath string

	Workdir  string
	Hostname string
}

func New() (*Facts, error) {
	f := new(Facts)

	f.Env = Env()

	if err := f.executable(); err != nil {
		return nil, err
	}

	if err := f.workdir(); err != nil {
		return nil, err
	}

	if err := f.hostname(); err != nil {
		return nil, err
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
