// downloadbasic sets up download test fixtures and runs all download cases.
package main

import (
	"context"
	"crypto"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/ntnn/tensile"
	"github.com/ntnn/tensile/nodes/std"
	"github.com/ntnn/tensile/pkg/app"
	"github.com/ntnn/tensile/pkg/queue"
)

// testContent is the file content served for download scenarios.
const testContent = "hello from download test\n"

// sha256Length is the expected length of a SHA-256 hex digest.
const sha256Length = 64

// reportFileMode is the permission for written report artifacts.
const reportFileMode = 0o600

// readHeaderTimeout is the timeout for reading HTTP request headers.
const readHeaderTimeout = 5 * time.Second

func main() {
	if err := run(context.Background()); err != nil {
		log.Fatal(err)
	}
}

func run(ctx context.Context) error {
	// start HTTP server
	server := startServer(ctx)
	defer func() {
		if err := server.Close(); err != nil {
			log.Printf("server close: %v", err)
		}
	}()

	baseURL := "http://" + server.Addr

	// main queue: download all files
	q := queue.New()
	q.Add(
		&std.Dir{Path: "/opt/e2e/download"},
		// basic download to file
		&std.Download{
			URL:    baseURL + "/testfile.txt",
			Output: "/opt/e2e/download/basic.txt",
		},
		// download to directory (filename inferred)
		&std.Download{
			URL:    baseURL + "/testfile.txt",
			Output: "/opt/e2e/download/dir-test/",
		},
		// download with explicit checksum
		&std.Download{
			URL:      baseURL + "/testfile.txt",
			Output:   "/opt/e2e/download/checksum.txt",
			Checksum: "7367cc55a330ebfa85453c7c433f94c570f47e7b52275ec05293bc52e0a221d5",
		},
		// download with checksum URL
		&std.Download{
			URL:         baseURL + "/testfile.txt",
			Output:      "/opt/e2e/download/checksum-url.txt",
			ChecksumURL: baseURL + "/testfile.txt.sha256",
		},
		// download with content-digest header
		&std.Download{
			URL:    baseURL + "/content-digest-file.txt",
			Output: "/opt/e2e/download/content-digest.txt",
		},
	)
	if err := app.New().Run(ctx, q); err != nil {
		return fmt.Errorf("download queue: %w", err)
	}

	// checksum fail: separate queue that expects error
	failQ := queue.New()
	failQ.Add(&std.Download{
		URL:      baseURL + "/testfile.txt",
		Output:   "/opt/e2e/download/checksum-fail.txt",
		Checksum: strings.Repeat("a", sha256Length),
	})
	failErr := app.New().Run(ctx, failQ)
	if failErr != nil {
		// expected: write error to file for test to check
		path := "/opt/e2e/download/checksum-fail-error.txt"
		if err := os.WriteFile(path, []byte(failErr.Error()), reportFileMode); err != nil {
			return fmt.Errorf("write fail error: %w", err)
		}
	}

	// absent: download then remove
	q3 := queue.New()
	q3.Add(&std.Download{
		URL:    baseURL + "/testfile.txt",
		Output: "/opt/e2e/download/absent.txt",
	})
	if err := app.New().Run(ctx, q3); err != nil {
		return fmt.Errorf("absent download: %w", err)
	}
	q4 := queue.New()
	q4.Add(&std.Download{
		State:  tensile.Absent,
		URL:    baseURL + "/testfile.txt",
		Output: "/opt/e2e/download/absent.txt",
	})
	if err := app.New().Run(ctx, q4); err != nil {
		return fmt.Errorf("absent remove: %w", err)
	}

	return nil
}

// startServer returns an HTTP server serving test files.
func startServer(ctx context.Context) *http.Server {
	mux := http.NewServeMux()

	mux.HandleFunc("/testfile.txt", func(w http.ResponseWriter, _ *http.Request) {
		if _, err := fmt.Fprint(w, testContent); err != nil {
			log.Printf("write testfile.txt: %v", err)
		}
	})

	h := crypto.SHA256.New()
	h.Write([]byte(testContent))
	sum := h.Sum(nil)

	hash := hex.EncodeToString(sum)
	mux.HandleFunc("/testfile.txt.sha256", func(w http.ResponseWriter, _ *http.Request) {
		if _, err := fmt.Fprintf(w, "%s  testfile.txt\n", hash); err != nil {
			log.Printf("write testfile.txt.sha256: %v", err)
		}
	})

	digest := base64.RawURLEncoding.EncodeToString(sum)
	mux.HandleFunc("/content-digest-file.txt", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Digest", "sha-256="+digest)
		if _, err := fmt.Fprint(w, testContent); err != nil {
			log.Printf("write content-digest-file.txt: %v", err)
		}
	})

	ln, err := (&net.ListenConfig{}).Listen(ctx, "tcp", "127.0.0.1:0")
	if err != nil {
		log.Fatal(err)
	}

	server := &http.Server{
		Handler:           mux,
		ReadHeaderTimeout: readHeaderTimeout,
	}
	go func() {
		if err := server.Serve(ln); err != nil && err != http.ErrServerClosed {
			log.Fatal(err)
		}
	}()
	server.Addr = ln.Addr().String()
	return server
}
