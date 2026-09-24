// A quick command to test things.
package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"log/slog"
	"os"
	"path/filepath"

	"github.com/ntnn/tensile/nodes/std"
	"github.com/ntnn/tensile/pkg/engine"
	"github.com/ntnn/tensile/pkg/queue"
)

func main() {
	if err := run(context.Background()); err != nil {
		log.Fatal(err)
	}
}

func run(ctx context.Context) error {
	fDebug := false
	fNoop := false
	flag.BoolVar(&fDebug, "debug", false, "enable debug logging")
	flag.BoolVar(&fNoop, "noop", false, "check only, do not modify")
	flag.Parse()

	q := queue.New()

	if fDebug {
		slog.SetLogLoggerLevel(slog.LevelDebug)
	}

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
		return err
	}
	defer os.RemoveAll(dir) //nolint:errcheck

	config := filepath.Join(dir, "config")
	if err := os.WriteFile(config, []byte("a=1\nb=2\nc=3\n"), 0o600); err != nil { //nolint:mnd // test fixture
		return err
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
	)

	work, err := q.Build()
	if err != nil {
		return err
	}

	seq := engine.NewSequential(
		work,
		engine.Options{
			Noop: fNoop,
		},
	)

	if err := seq.Execute(ctx); err != nil {
		return err
	}

	summary := seq.Summary()
	slog.InfoContext(ctx, "run finished", "summary", summary)
	fmt.Println(summary)

	return nil
}
