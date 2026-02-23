package main

import (
	"fmt"
	"log"

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
	q := queue.New()

	if err := q.Enqueue(
		&tensilestd.Print{
			Message: "Hello, %s!",
			Args:    []any{"world"},
		},
	); err != nil {
		return err
	}

	work, err := q.Build()
	if err != nil {
		return err
	}

	seq := engine.NewSequential(
		work,
		engine.Options{
			Noop: true,
		},
	)

	if err := seq.Execute(); err != nil {
		return err
	}

	summary := seq.Summary()
	fmt.Printf("Execution summary: %+v\n", summary)

	return nil
}
