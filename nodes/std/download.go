package std

import (
	"bytes"
	"context"
	"crypto"
	"encoding/base64"
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
	"github.com/ntnn/tensile/pkg/queue"
)

var (
	_ tensile.Identifier = (*Download)(nil)
	_ tensile.Validator  = (*Download)(nil)
	_ tensile.Depender   = (*Download)(nil)
	_ tensile.Executor   = (*Download)(nil)
	_ queue.NamedQueuer  = (*Download)(nil)
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

	Chown
	Chmod

	// TODO(ntnn): reporting
}

// DownloadIdentity returns the identity of the node downloading from downloadURL to path.
func DownloadIdentity(downloadURL string, path string) tensile.Identity {
	return tensile.AsIdentity("download", "url", downloadURL, "path", path)
}

// DownloadQueueIdentity returns the identity of the named queue for a Download node.
func DownloadQueueIdentity(path string) tensile.Identity {
	return tensile.AsIdentity("queue", "download", path)
}

// Identity implements [tensile.Identifier].
func (d *Download) Identity() tensile.Identity {
	return DownloadIdentity(d.URL, d.output())
}

// Conflicts implements [tensile.Conflictor].
func (d *Download) Conflicts() ([]tensile.Identity, error) {
	return []tensile.Identity{FileIdentity(d.output())}, nil
}

// DependsOn implements [tensile.Depender].
func (d *Download) DependsOn() ([]tensile.Identity, error) {
	return ParentDirIdentities(d.output()), nil
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

func hashFromChecksum(checksum string) (crypto.Hash, error) {
	switch len(checksum) {
	case 32:
		return crypto.MD5, nil
	case 40:
		return crypto.SHA1, nil
	case 64:
		return crypto.SHA256, nil
	case 128:
		return crypto.SHA512, nil
	default:
		return 0, fmt.Errorf("unable to detect hash algorithm from checksum length %d", len(checksum))
	}
}

func (d *Download) Queue() *queue.NamedQueue {
	q := queue.NewNamed(DownloadQueueIdentity(d.output()))

	q.Add(tensile.NewNode(d))
	if d.state() == tensile.Absent {
		return q
	}

	chown := d.Chown
	chown.Path = d.output()
	q.Add(&chown)

	chmod := d.Chmod
	chmod.Path = d.output()
	q.Add(&chmod)

	return q
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
		algorithm, expected, err := d.expectedChecksum(wire.Context())
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

		localHash, err := computeFileHash(path, algorithm)
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
func (d *Download) Execute(wire tensile.Wire) (tensile.Diff, error) {
	switch d.state() {
	case tensile.Absent:
		if err := os.Remove(d.output()); err != nil && !errors.Is(err, os.ErrNotExist) {
			return nil, fmt.Errorf("remove: %w", err)
		}
		return diff.NewFieldChanges(&diff.FieldChange{
			Field: "file",
			Old:   string(tensile.Present),
			New:   diff.Absent,
		}), nil

	case tensile.ReadOnly:
		return nil, nil //nolint:nilnil

	case tensile.Present:
		path := d.output()

		// ensure parent dir exists
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil { //nolint:mnd,gosec
			return nil, fmt.Errorf("mkdir: %w", err)
		}

		// download to temp file
		tmpFile, err := os.CreateTemp(filepath.Dir(path), ".download-*")
		if err != nil {
			return nil, fmt.Errorf("create temp: %w", err)
		}
		tmpPath := tmpFile.Name()
		defer func() {
			// TODO(ntnn): Add a TempDir to wire that is cleaned up after a run.
			// Similar to testing.T.TempDir
			if tmpPath != "" {
				os.Remove(tmpPath)
			}
		}()

		if err := download(wire.Context(), d.URL, tmpFile); err != nil {
			tmpFile.Close()
			return nil, fmt.Errorf("downloading file: %w", err)
		}
		if err := tmpFile.Close(); err != nil {
			return nil, fmt.Errorf("close temp: %w", err)
		}

		// verify checksum before moving
		algorithm, expected, err := d.expectedChecksum(wire.Context())
		if err != nil {
			return nil, err
		}
		if expected != "" {
			localHash, err := computeFileHash(tmpPath, algorithm)
			if err != nil {
				return nil, err
			}
			if !strings.EqualFold(localHash, expected) {
				return nil, fmt.Errorf("checksum mismatch on downloaded file: expected %s, got %s", expected, localHash)
			}
		}

		localHash := diff.Absent
		if algorithm != 0 {
			var err error
			localHash, err = computeFileHash(path, algorithm)
			if err != nil && !errors.Is(err, os.ErrNotExist) {
				return nil, fmt.Errorf("calculating current hash: %w", err)
			}
		}

		// move to final destination
		if err := os.Rename(tmpPath, path); err != nil {
			return nil, fmt.Errorf("rename: %w", err)
		}
		tmpPath = "" // prevent deferred cleanup
		return diff.NewFieldChanges(&diff.FieldChange{
			Field: "file",
			Old:   localHash,
			New:   expected,
		}), nil

	default:
		return nil, errors.New("unreachable")
	}
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

func (d *Download) expectedChecksum(ctx context.Context) (crypto.Hash, string, error) {
	switch {
	case d.Checksum != "":
		checksum := strings.TrimSpace(d.Checksum)
		algorithm, err := hashFromChecksum(checksum)
		if err != nil {
			return 0, "", err
		}
		return algorithm, checksum, nil
	case d.ChecksumURL != "":
		checksum, err := d.checksumFromURL(ctx)
		if err != nil {
			return 0, "", err
		}
		algorithm, err := hashFromChecksum(checksum)
		if err != nil {
			return 0, "", err
		}
		return algorithm, checksum, nil
	default:
		// Try Content-Digest header
		resp, err := headRequest(ctx, d.URL)
		if err != nil {
			return 0, "", fmt.Errorf("getting Content-Digest: %w", err)
		}
		defer resp.Body.Close()

		digestHeader := resp.Header.Get("Content-Digest")
		if digestHeader == "" {
			return 0, "", nil // no Content-Digest header
		}

		algorithm, checksum, err := parseContentDigest(digestHeader)
		if err != nil {
			return 0, "", fmt.Errorf("parsing Content-Digest header: %w", err)
		}

		return algorithm, checksum, nil
	}
}

func download(ctx context.Context, from string, to io.Writer) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, from, nil)
	if err != nil {
		return fmt.Errorf("building request: %w", err)
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return fmt.Errorf("getting url: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("url returned %d", resp.StatusCode)
	}

	if _, err := io.Copy(to, resp.Body); err != nil {
		return fmt.Errorf("copy body to buffer: %w", err)
	}

	return nil
}

func headRequest(ctx context.Context, location string) (*http.Response, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodHead, location, nil)
	if err != nil {
		return nil, fmt.Errorf("building request: %w", err)
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("getting url: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		resp.Body.Close()
		return nil, fmt.Errorf("url returned %d", resp.StatusCode)
	}

	return resp, nil
}

func (d *Download) checksumFromURL(ctx context.Context) (string, error) {
	buf := &bytes.Buffer{}
	if err := download(ctx, d.ChecksumURL, buf); err != nil {
		return "", fmt.Errorf("error downloading checksum file: %w", err)
	}

	u, err := url.Parse(d.URL)
	if err != nil {
		return "", fmt.Errorf("invalid URL: %w", err)
	}

	filename := filepath.Base(u.Path)
	if filename == "" || filename == "." {
		return "", fmt.Errorf("URL does not end in a filename to match from ChecksumURL: %q", d.URL)
	}

	for {
		line, err := buf.ReadString('\n')
		if errors.Is(err, io.EOF) {
			return "", fmt.Errorf("file at ChecksumURL %q does not contain hash for filename %q", d.ChecksumURL, filename)
		}
		if err != nil {
			return "", fmt.Errorf("error reading line from buffered checksum file: %w", err)
		}

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
}

func hashFromDigestName(name string) (crypto.Hash, error) {
	switch name {
	case "md5":
		return crypto.MD5, nil
	case "sha-1", "sha1":
		return crypto.SHA1, nil
	case "sha-256", "sha256":
		return crypto.SHA256, nil
	case "sha-512", "sha512":
		return crypto.SHA512, nil
	default:
		return 0, fmt.Errorf("unknown digest algorithm %q", name)
	}
}

// parseContentDigest parses the Content-Digest header value.
func parseContentDigest(headerValue string) (crypto.Hash, string, error) {
	// Header format: sha-256=abcdef... or multiple comma-separated
	for part := range strings.SplitSeq(headerValue, ",") {
		part = strings.TrimSpace(part)
		name, value, ok := strings.Cut(part, "=")
		if !ok {
			continue
		}

		algorithm, err := hashFromDigestName(strings.TrimSpace(name))
		if err != nil {
			continue
		}

		// Digest is base64url encoded
		raw, err := base64.RawURLEncoding.DecodeString(strings.TrimSpace(value))
		if err != nil {
			return 0, "", fmt.Errorf("decoding digest: %w", err)
		}

		return algorithm, hex.EncodeToString(raw), nil
	}

	return 0, "", errors.New("no supported digest found in Content-Digest header")
}
