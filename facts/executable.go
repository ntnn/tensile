package facts

import (
	"os"
	"path/filepath"
)

// Returns the base name and the fully qualified path of the running
// binary.
func Executable() (string, string, error) {
	full, err := os.Executable()
	if err != nil {
		return "", "", err
	}

	return filepath.Base(full), full, nil
}

func (f *Facts) executable() error {
	var err error
	f.Executable, f.ExecutablePath, err = Executable()
	return err
}
