// packagepacman installs and removes packages through pacman.
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

	// already installed, must be a no-op
	preinstalled := &std.Package{Name: "gzip"}
	// not installed, must be installed
	install := &std.Package{Name: "tree"}
	install2 := &std.Package{Name: "jc"}
	install3 := &std.Package{Name: "bc"}
	// installed by the test beforehand, must be removed
	remove := &std.Package{Name: "less", State: std.PackageAbsent}
	// not installed, must stay absent as a no-op
	absent := &std.Package{Name: "jq", State: std.PackageAbsent}

	if err := q.Add(preinstalled, install, install2, install3, remove, absent); err != nil {
		return err
	}

	work, err := q.Build()
	if err != nil {
		return err
	}

	par := engine.NewParallel(work, engine.ParallelOptions{})
	if err := par.Execute(ctx); err != nil {
		return err
	}

	return nil
}
