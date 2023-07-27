//go:build !windows

package facts

import (
	"bufio"
	"errors"
	"fmt"
	"os"
	"reflect"
	"strings"
)

const ProbablyContainer = "probably-container"

func NewOSRelease() (OSRelease, error) {
	rel, err := newOSReleaseFile()
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		return OSRelease{}, fmt.Errorf("error reading info from os-release file: %w", err)
	}
	if err == nil {
		return rel, nil
	}

	// not returning an error - the various container envrionments are
	// too difficult to reliably detect.
	// Instead OSRelease.Name and .ID are set to ProbablyContainer.
	return OSRelease{
		Name: ProbablyContainer,
		ID:   ProbablyContainer,
	}, nil
}

func newOSReleaseFile() (OSRelease, error) {
	for _, path := range []string{"/etc/os-release", "/usr/lib/os-release"} {
		rel, err := newOSReleaseFileWithPath(path)
		if err != nil && !errors.Is(err, os.ErrNotExist) {
			return OSRelease{}, err
		}
		if err == nil {
			return rel, nil
		}
	}
	return OSRelease{}, os.ErrNotExist
}

func newOSReleaseFileWithPath(path string) (OSRelease, error) {
	f, err := os.Open(path)
	if err != nil {
		return OSRelease{}, err
	}
	defer f.Close()

	rel := OSRelease{}
	rel.IDLike = []string{}

	relValue := reflect.ValueOf(&rel)
	relType := reflect.TypeOf(rel)

	scanner := bufio.NewScanner(f)

	for scanner.Scan() {
		// VARIABLE=VALUE
		split := strings.SplitN(scanner.Text(), "=", 2)
		key := split[0]
		value := ""
		if len(split) == 2 {
			value = split[1]
		}

		// Remove quotations
		if len(value) >= 2 {
			if value[0] == '"' && value[len(value)-1] == '"' {
				value = value[1 : len(value)-1]
			}
		}

		for i := 0; i < relType.NumField(); i++ {
			field := relType.Field(i)
			if key != field.Tag.Get("osrel") {
				continue
			}

			fieldValue := relValue.Elem().FieldByName(field.Name)
			switch fieldValue.Kind() {
			case reflect.String:
				fieldValue.SetString(value)
			case reflect.Slice:
				fieldValue.Set(reflect.ValueOf(strings.Split(value, " ")))
			default:
				return OSRelease{}, fmt.Errorf("unhandled field kind %q", fieldValue.Kind())
			}
		}
	}

	if err := scanner.Err(); err != nil {
		return OSRelease{}, err
	}

	return rel, nil
}
