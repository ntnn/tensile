package main

import (
	"flag"
	"fmt"
	"log"
	"log/slog"

	"github.com/ntnn/tensile/pkg/engine"
	"github.com/ntnn/tensile/pkg/queue"
	"github.com/ntnn/tensile/tensilestd"
)

func main() {
	if err := run(); err != nil {
		log.Fatal(err)
	}
}

func run() error {
	fDebug := false
	flag.BoolVar(&fDebug, "debug", false, "enable debug logging")
	flag.Parse()

	q := queue.New()

	if fDebug {
		slog.SetLogLoggerLevel(slog.LevelDebug)
	}

	print1 := &tensilestd.Print{
		Message: "Hello, %s!",
		Args:    []any{"world"},
	}

	print2 := &tensilestd.Print{
		Message: "The answer is %d.",
		Args:    []any{42},
	}

	if err := q.Enqueue(print1, print2); err != nil {
		return err
	}

	q.Depends(print1, print2)

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

	if err := seq.Execute(); err != nil {
		return err
	}

	summary := seq.Summary()
	fmt.Printf("Execution summary: %+v\n", summary)

	return nil
}
