// openwrtbasic exercises core std nodes on an OpenWrt machine.
package main

import (
	"context"
	"log"

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
	cmd := &std.Command{
		Command: "touch /opt/e2e/command-ran",
		Creates: "/opt/e2e/command-ran",
	}

	if err := q.Add(dir, file, cmd); err != nil {
		return err
	}
	if err := q.DependsOn(cmd, dir); err != nil {
		return err
	}

	work, err := q.Build()
	if err != nil {
		return err
	}

	seq := engine.NewParallel(work, engine.ParallelOptions{})
	return seq.Execute(ctx)
}
