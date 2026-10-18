package std

import (
	"context"
	"crypto"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"

	"github.com/ntnn/tensile"
	"github.com/ntnn/tensile/pkg/diff"
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

func (d *Download) validateAlgorithm() error {
	switch d.Algorithm {
	case "", "sha256", "sha512", "sha1", "md5":
		return nil
	}
	return fmt.Errorf("unknown Algorithm %q", d.Algorithm)
}

func (d *Download) algorithm() crypto.Hash {
	switch d.Algorithm {
	case "sha1":
		return crypto.SHA1
	case "sha256":
		return crypto.SHA256
	case "sha512":
		return crypto.SHA512
	case "md5":
		return crypto.MD5
	default:
		return crypto.SHA256
	}
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

// NeedsExecution implements [tensile.Executor].
func (d *Download) NeedsExecution(wire tensile.Wire) (bool, tensile.Diff, error) {
	path := d.output()

	switch d.state() {
	case tensile.Absent:
		_, err := os.Stat(path)
		if errors.Is(err, os.ErrNotExist) {
			return false, nil, nil
		}
		if err != nil {
			return false, nil, fmt.Errorf("stat: %w", err)
		}
		return true, diff.NewFieldChanges(&diff.FieldChange{
			Field: "file",
			Old:   string(tensile.Present),
			New:   diff.Absent,
		}), nil

	case tensile.ReadOnly:
		// always report
		return true, nil, nil

	case tensile.Present:
		// check existence
		_, err := os.Stat(path)
		if errors.Is(err, os.ErrNotExist) {
			return true, diff.NewFieldChanges(&diff.FieldChange{
				Field: "file",
				Old:   diff.Absent,
				New:   string(tensile.Present),
			}), nil
		}
		if err != nil {
			return false, nil, fmt.Errorf("stat: %w", err)
		}

		// file exists, check checksum if provided
		expected, err := d.expectedChecksum(wire.Context())
		if err != nil {
			return false, nil, err
		}

		if expected == "" {
			// no checksum: fall back to reporting change
			return true, diff.NewFieldChanges(&diff.FieldChange{
				Field: "file",
				Old:   string(tensile.Present),
				New:   "assuming update, no checksum configured",
			}), nil
		}

		localHash, err := computeFileHash(path, d.algorithm())
		if err != nil {
			return false, nil, err
		}

		if strings.EqualFold(localHash, expected) {
			return false, nil, nil
		}
		return true, diff.NewFieldChanges(&diff.FieldChange{
			Field: "checksum",
			Old:   localHash,
			New:   expected,
		}), nil

	default:
		return false, nil, errors.New("unreachable")
	}
}

// Execute implements [tensile.Executor].
func (d *Download) Execute(_ tensile.Wire) (tensile.Diff, error) {
	// implementation
	return nil, nil //nolint:nilnil
}

func computeFileHash(path string, hash crypto.Hash) (string, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", fmt.Errorf("open: %w", err)
	}
	defer f.Close()

	h := hash.New()
	if _, err := io.Copy(h, f); err != nil {
		return "", fmt.Errorf("hash: %w", err)
	}

	return hex.EncodeToString(h.Sum(nil)), nil
}

func (d *Download) expectedChecksum(ctx context.Context) (string, error) {
	switch {
	case d.Checksum != "":
		return strings.TrimSpace(d.Checksum), nil
	case d.ChecksumURL != "":
		return d.checksumFromURL(ctx)
	default:
		// TODO(ntnn): implement content-digest heast
		return "", nil
	}
}

func (d *Download) checksumFromURL(ctx context.Context) (string, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", d.ChecksumURL, nil)
	if err != nil {
		return "", fmt.Errorf("building request to get checksum file: %w", err)
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("getting ChecksumURL: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("ChecksumURL returned %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("read ChecksumURL body: %w", err)
	}

	u, err := url.Parse(d.URL)
	if err != nil {
		return "", fmt.Errorf("invalid URL: %w", err)
	}

	filename := filepath.Base(u.Path)
	if filename == "" || filename == "." {
		return "", fmt.Errorf("URL does not end in a filename to match from ChecksumURL: %q", d.URL)
	}

	for _, line := range strings.Split(string(body), "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		// hashsum format: <hash>  <filename> or <hash> <filename>
		parts := strings.Fields(line)
		if len(parts) < 2 {
			continue
		}
		if parts[1] == filename {
			return parts[0], nil
		}
	}

	return "", fmt.Errorf("file at ChecksumURL %q does not contain hash for filename %q", d.ChecksumURL, filename)
}
