package nodes

import (
	"bytes"
	"crypto/md5"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"os"
	"path/filepath"

	"github.com/ntnn/tensile"
)

var _ tensile.Node = (*File)(nil)

type File struct {
	Target  string
	Content string

	dirs []string
}

func (file *File) Validate() error {
	if file.Target == "" {
		return fmt.Errorf("target cannot be empty")
	}

	dirs, err := walkDirs(file.Target)
	if err != nil {
		return err
	}
	file.dirs = dirs

	return nil
}

func walkDirs(target string) ([]string, error) {
	last := filepath.FromSlash(target)
	ret := []string{}

	for s := filepath.Dir(target); s != last; s = filepath.Dir(s) {
		// 1. target=/ -> no dependency
		// 2. target=/a -> []string{"/"}
		if s == target || s == last || s == filepath.VolumeName(s) {
			break
		}
		ret = append(ret, tensile.FormatIdentity(tensile.Path, s))
		last = s
	}

	return ret, nil
}

func (file File) Shape() tensile.Shape {
	return tensile.Path
}

func (file File) Identifier() string {
	return file.Target
}

var _ tensile.AfterNoder = (*File)(nil)

func (file File) AfterNodes() []string {
	return file.dirs
}

func (file File) sourceHash() []byte {
	sum := md5.Sum([]byte(file.Content))
	return sum[:]
}

func (file File) targetHash() ([]byte, error) {
	f, err := os.Open(file.Target)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	h := md5.New()
	if _, err := io.Copy(h, f); err != nil {
		return nil, err
	}

	return []byte(h.Sum(nil)), nil
}

func (file File) NeedsExecution(ctx tensile.Context) (bool, error) {
	targetHash, err := file.targetHash()
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			// file does not exist, must be created
			return true, nil
		}
		return false, fmt.Errorf("tensile: error calculating hash of target %q: %w", file.Target, err)
	}

	sourceHash := file.sourceHash()

	ctx.Logger().Debug("comparing hashes",
		slog.Any("source", sourceHash),
		slog.Any("target", targetHash),
	)

	return bytes.Compare(file.sourceHash(), targetHash) != 0, nil
}

func (file File) Execute(ctx tensile.Context) (any, error) {
	f, err := os.Create(file.Target)
	if err != nil {
		return nil, fmt.Errorf("tensile: error opening and truncating target %q: %w", file.Target, err)
	}
	defer f.Close()

	if _, err := f.WriteString(file.Content); err != nil {
		return nil, fmt.Errorf("tensile: error writing content to target %q: %w", file.Target, err)
	}

	return nil, nil
}
