package std

import (
	"errors"
	"fmt"
	"net/url"
	"path/filepath"
	"strings"

	"github.com/ntnn/tensile"
)

var (
	_ tensile.Identifier = (*Download)(nil)
	_ tensile.Validator  = (*Download)(nil)
	_ tensile.Executor   = (*Download)(nil)
)

// Download ensures a remote file is downloaded and optionally verified.
type Download struct {
	// State is the desired existence.
	// Defaults to [tensile.Present].
	State tensile.State

	// URL is the remote file location.
	// Required.
	URL string
	// Output is the local destination.
	// May be a file path or a directory (ending in /).
	// Required.
	Output string

	// Checksum is the expected hash of the file.
	// Mutually exclusive with ChecksumURL.
	Checksum string
	// ChecksumURL is a URL to a hashsum file containing the expected hash.
	// Mutually exclusive with Checksum.
	ChecksumURL string

	// Algorithm specifies the hash algorithm.
	// Defaults to "sha256" or is detected from hash length.
	Algorithm string
}

// DownloadIdentity returns the identity of the node downloading from downloadURL to path.
func DownloadIdentity(downloadURL string, path string) tensile.Identity {
	return tensile.AsIdentity("download", "url", downloadURL, "path", path)
}

// Identity implements [tensile.Identifier].
func (d *Download) Identity() tensile.Identity {
	return DownloadIdentity(d.URL, d.output())
}

func (d *Download) state() tensile.State {
	return d.State.OrDefault()
}

func (d *Download) output() string {
	if !strings.HasSuffix(d.Output, "/") {
		return d.Output
	}

	u, err := url.Parse(d.URL)
	if err != nil {
		return d.Output
	}

	name := filepath.Base(u.Path)
	if name == "" || name == "." {
		name = "download"
	}

	return d.Output + name
}

// Validate implements [tensile.Validator].
func (d *Download) Validate(_ tensile.Wire) error {
	if d.URL == "" {
		return errors.New("URL is required")
	}
	if _, err := url.Parse(d.URL); err != nil {
		return fmt.Errorf("invalid URL: %w", err)
	}

	if d.Output == "" {
		return errors.New("Output is required")
	}

	if d.Checksum != "" && d.ChecksumURL != "" {
		return errors.New("Checksum and ChecksumURL are mutually exclusive")
	}

	if err := d.validateAlgorithm(); err != nil {
		return err
	}

	if err := d.state().Valid(); err != nil {
		return err
	}
	return nil
}

func (d *Download) validateAlgorithm() error {
	switch d.Algorithm {
	case "", "sha256", "sha512", "sha1", "md5":
		return nil
	}
	return fmt.Errorf("unknown Algorithm %q", d.Algorithm)
}

// NeedsExecution implements [tensile.Executor].
func (d *Download) NeedsExecution(_ tensile.Wire) (bool, tensile.Diff, error) {
	// implementation
	return false, nil, nil //nolint:nilnil
}

// Execute implements [tensile.Executor].
func (d *Download) Execute(_ tensile.Wire) (tensile.Diff, error) {
	// implementation
	return nil, nil //nolint:nilnil
}
