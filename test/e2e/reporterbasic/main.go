// reporterbasic exercises node outputs
package main

import (
	"context"
	"fmt"
	"log"
	"os"

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
		Path:    "/opt/e2e/reporter-file.txt",
		Content: "hello from reporter\n",
	}
	cmd := &std.Command{
		Command: "echo hello reporter",
	}

	fileConsumer := &writeReport{
		Path: "/opt/e2e/reporter-hash",
		Dep:  file.Identity(),
		Read: func(wire tensile.Wire) (string, error) {
			out, err := wire.Storage().Get[std.FileContentOutput](file.Identity())
			if err != nil {
				return "", err
			}
			return out.SHA256, nil
		},
	}
	cmdConsumer := &writeReport{
		Path: "/opt/e2e/reporter-cmd",
		Dep:  cmd.Identity(),
		Read: func(wire tensile.Wire) (string, error) {
			out, err := wire.Storage().Get[std.CommandOutput](cmd.Identity())
			if err != nil {
				return "", err
			}
			return out.Output, nil
		},
	}

	if err := q.Enqueue(dir, file, cmd, fileConsumer, cmdConsumer); err != nil {
		return err
	}

	work, err := q.Build()
	if err != nil {
		return err
	}

	seq := engine.NewParallel(work, engine.ParallelOptions{})
	return seq.Execute(ctx)
}

// reportFileMode is the permission for written report artifacts.
const reportFileMode = 0o644

// writeReport writes the value read from a dependency's output to Path.
type writeReport struct {
	Path string
	Dep  tensile.Identity
	Read func(wire tensile.Wire) (string, error)
}

var _ tensile.Identifier = (*writeReport)(nil)
var _ tensile.Depender = (*writeReport)(nil)
var _ tensile.Executor = (*writeReport)(nil)

func (w *writeReport) Identity() tensile.Identity {
	return tensile.AsIdentity("writeReport", "path", w.Path)
}

func (w *writeReport) DependsOn() ([]tensile.Identity, error) {
	return []tensile.Identity{w.Dep}, nil
}

func (w *writeReport) NeedsExecution(_ tensile.Wire) (bool, error) {
	return true, nil
}

func (w *writeReport) Execute(wire tensile.Wire) error {
	value, err := w.Read(wire)
	if err != nil {
		return fmt.Errorf("reading output of %s: %w", w.Dep, err)
	}
	if err := os.WriteFile(w.Path, []byte(value), reportFileMode); err != nil {
		return fmt.Errorf("writing report to %s: %w", w.Path, err)
	}
	return nil
}
