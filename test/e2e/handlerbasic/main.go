// handlerbasic wires handlers to a file node to exercise notify semantics.
package main

import (
	"context"
	"log"

	"github.com/ntnn/tensile"
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
	q := queue.New()

	dir := &std.Dir{Path: "/opt/e2e"}
	file := &std.FileContent{
		Path:    "/opt/e2e/hello.txt",
		Content: "hello from tensile\n",
	}

	notified := tensile.NewHandler(&std.Command{
		Command: "touch /opt/e2e/notified",
	})

	// no notifiers, must never run
	silent := tensile.NewHandler(&std.Command{
		Command: "touch /opt/e2e/silent",
	})

	if err := q.Enqueue(dir, file, notified, silent); err != nil {
		return err
	}
	if err := q.NotifiedBy(notified, file); err != nil {
		return err
	}

	work, err := q.Build()
	if err != nil {
		return err
	}

	seq := engine.NewParallel(work, engine.ParallelOptions{})
	return seq.Execute(ctx)
}
