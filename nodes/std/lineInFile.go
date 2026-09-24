package std

import (
	"errors"
	"fmt"
	"os"
	"regexp"
	"strings"

	"github.com/ntnn/tensile"
)

var _ tensile.Identifier = (*LineInFile)(nil)
var _ tensile.Validator = (*LineInFile)(nil)
var _ tensile.Depender = (*LineInFile)(nil)
var _ tensile.Executor = (*LineInFile)(nil)

// LineInFile ensures a file contains Line.
// The last line matching Regexp is replaced with Line, otherwise Line is appended.
// A missing file is created.
type LineInFile struct {
	Path string
	// Regexp locates the line to replace.
	// Empty matches Line verbatim.
	Regexp string
	Line   string
}

// Identity implements [tensile.Identifier].
func (l *LineInFile) Identity() tensile.Identity {
	return tensile.AsIdentity("lineInFile", "path", l.Path, "regexp", l.Regexp)
}

// Validate implements [tensile.Validator].
func (l *LineInFile) Validate(_ tensile.Wire) error {
	if l.Path == "" {
		return errors.New("path is required")
	}
	if l.Line == "" {
		return errors.New("line is required")
	}
	if _, err := regexp.Compile(l.Regexp); err != nil {
		return fmt.Errorf("compiling Regexp: %w", err)
	}
	return nil
}

// DependsOn implements [tensile.Depender].
func (l *LineInFile) DependsOn() ([]tensile.Identity, error) {
	return append(ParentDirIdentities(l.Path), FileIdentity(l.Path)), nil
}

// NeedsExecution implements [tensile.Executor].
func (l *LineInFile) NeedsExecution(_ tensile.Wire) (bool, tensile.Diff, error) {
	content, err := l.read()
	if err != nil {
		return false, nil, err
	}

	result, err := l.apply(content)
	if err != nil {
		return false, nil, err
	}
	return result != content, nil, nil
}

// Execute implements [tensile.Executor].
func (l *LineInFile) Execute(_ tensile.Wire) (tensile.Diff, error) {
	content, err := l.read()
	if err != nil {
		return nil, err
	}

	result, err := l.apply(content)
	if err != nil {
		return nil, err
	}

	// perms apply only on create, existing files keep theirs
	if err := os.WriteFile(l.Path, []byte(result), 0o644); err != nil { //nolint:gosec,mnd
		return nil, fmt.Errorf("writing file: %w", err)
	}

	return nil, nil //nolint:nilnil // nil Diff is valid
}

// read returns the file content, empty when the file is missing.
func (l *LineInFile) read() (string, error) {
	content, err := os.ReadFile(l.Path)
	if os.IsNotExist(err) {
		return "", nil
	}
	if err != nil {
		return "", fmt.Errorf("reading file: %w", err)
	}
	return string(content), nil
}

// apply returns content with Line replacing the last match of Regexp,
// or appended when nothing matches.
func (l *LineInFile) apply(content string) (string, error) {
	re, err := regexp.Compile(l.Regexp)
	if err != nil {
		return "", fmt.Errorf("compiling Regexp: %w", err)
	}

	lines := strings.Split(strings.TrimSuffix(content, "\n"), "\n")
	if content == "" {
		lines = nil
	}

	match := -1
	for i, line := range lines {
		if l.Regexp == "" {
			if line == l.Line {
				match = i
			}
			continue
		}
		if re.MatchString(line) {
			match = i
		}
	}

	if match >= 0 {
		lines[match] = l.Line
	} else {
		lines = append(lines, l.Line)
	}

	return strings.Join(lines, "\n") + "\n", nil
}
