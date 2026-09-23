// A quick command to test things.
package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"log/slog"

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
	flag.BoolVar(&fDebug, "debug", false, "enable debug logging")
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

	if err := q.Add(print1, print2); err != nil {
		return err
	}

	if err := q.DependsOn(print1, print2); err != nil {
		return err
	}

	work, err := q.Build()
	if err != nil {
		return err
	}

	seq := engine.NewSequential(
		work,
		engine.Options{
			Noop: false,
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
