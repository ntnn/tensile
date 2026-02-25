package tensilestd

import "os"

type Chmod struct {
	Path     string
	FileMode os.FileMode
}

func (c Chmod) DependsOn() []string {
	return parentDirs(c.Path)
}

func (c Chmod) NeedsExecution() (bool, error) {
	info, err := os.Stat(c.Path)
	if err != nil {
		return false, err
	}
	return info.Mode() != c.FileMode, nil
}

func (c Chmod) Execute() error {
	return os.Chmod(c.Path, c.FileMode)
}
