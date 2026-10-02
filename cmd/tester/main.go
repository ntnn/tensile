// A quick command to test things.
package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"

	"github.com/ntnn/tensile/nodes/std"
	"github.com/ntnn/tensile/pkg/app"
	"github.com/ntnn/tensile/pkg/queue"
)

func main() {
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	err := run(ctx)
	cancel()
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
}

func run(ctx context.Context) error {
	q, fn, err := build()
	if err != nil {
		return err
	}
	defer fn()

	a := app.New()
	a.AddFlags(flag.CommandLine)
	flag.Parse()
	return a.Run(ctx, q)
}

func build() (*queue.Queue, func(), error) {
	q := queue.New()

	print1 := &std.Print{
		Message: "Hello, %s!",
		Args:    []any{"world"},
	}

	print2 := &std.Print{
		Message: "The answer is %d.",
		Args:    []any{42},
	}

	q.Add(print1, print2)
	q.DependsOn(print1, print2)

	dir, err := os.MkdirTemp("", "tensile-tester-")
	if err != nil {
		return nil, nil, err
	}
	fn := func() { os.RemoveAll(dir) } //nolint:errcheck,gosec

	config := filepath.Join(dir, "config")
	if err := os.WriteFile(config, []byte("a=1\nb=2\nc=3\n"), 0o600); err != nil { //nolint:mnd // test fixture
		return nil, fn, err
	}

	satisfied := filepath.Join(dir, "satisfied")
	if err := os.WriteFile(satisfied, []byte("done\n"), 0o600); err != nil { //nolint:mnd // test fixture
		return nil, fn, err
	}

	q.Add(
		&std.FileContent{
			Path:    config,
			Content: "a=1\nb=5\nc=3\nd=4\n",
		},
		&std.LineInFile{
			Path:   config,
			Regexp: "^b=",
			Line:   "b=6",
		},
		&std.FileContent{
			Path:    satisfied,
			Content: "done\n",
		},
	)

	return q, fn, nil
}
