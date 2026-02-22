package tensilestd

import (
	"fmt"
	"os"
)

type File struct {
	Chmod
	Chown

	Path    string
	Content string
}

func (f *File) Validate() error {
	f.Chmod.Path = f.Path
	f.Chown.Path = f.Path
	return nil
}

func (f *File) Provides() ([]string, error) {
	return []string{f.Path}, nil
}

func (f *File) DependsOn() ([]string, error) {
	// TODO parent dirs
	return []string{}, nil
}

func (f *File) NeedsExecution() (bool, error) {
	chmodNeeded, err := f.Chmod.NeedsExecution()
	if err != nil {
		return false, fmt.Errorf("error checking chmod: %w", err)
	}
	if chmodNeeded {
		return true, nil
	}

	chownNeeded, err := f.Chown.NeedsExecution()
	if err != nil {
		return false, fmt.Errorf("error checking chown: %w", err)
	}
	if chownNeeded {
		return true, nil
	}

	info, err := os.Stat(f.Path)
	if err != nil {
		return false, fmt.Errorf("error stating file: %w", err)
	}

	// TODO compare content
	return info.Size() != int64(len(f.Content)), nil
}

func (f *File) Execute() error {
	fd, err := os.Create(f.Path)
	if err != nil {
		return fmt.Errorf("error creating file: %w", err)
	}
	defer fd.Close()

	if _, err := fd.WriteString(f.Content); err != nil {
		return fmt.Errorf("error writing to file: %w", err)
	}

	if err := f.Chmod.Execute(); err != nil {
		return fmt.Errorf("error setting file permissions: %w", err)
	}

	if err := f.Chown.Execute(); err != nil {
		return fmt.Errorf("error setting file ownership: %w", err)
	}

	return nil
}
