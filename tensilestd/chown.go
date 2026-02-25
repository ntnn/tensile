package tensilestd

import "os"

type Chown struct {
	Path  string
	Owner string
	Group string
}

func (c Chown) DependsOn() []string {
	return parentDirs(c.Path)
}

func (c Chown) NeedsExecution() (bool, error) {
	// TODO resolve owner and group names to numeric IDs
	// TODO check if the current owner and group match the desired ones
	return true, nil
}

func (c Chown) Execute() error {
	// TODO resolve owner and group names to numeric IDs
	return os.Chown(c.Path, -1, -1)
}
