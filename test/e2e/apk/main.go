// apk installs and removes packages through apk.
package main

import (
	"context"
	"log"

	"github.com/ntnn/tensile"
	"github.com/ntnn/tensile/nodes/std"
	"github.com/ntnn/tensile/pkg/app"
	"github.com/ntnn/tensile/pkg/queue"
)

func main() {
	if err := run(context.Background()); err != nil {
		log.Fatal(err)
	}
}

func run(ctx context.Context) error {
	q := queue.New()

	update := &std.PackageManagerUpdate{}

	// already installed, must be a no-op
	preinstalled := &std.Package{Name: "busybox"}
	// not installed, must be installed
	install := &std.Package{Name: "tree"}
	// installed by the test beforehand, must be removed
	remove := &std.Package{Name: "less", State: tensile.Absent}
	// not installed, must stay absent as a no-op
	absent := &std.Package{Name: "jq", State: tensile.Absent}

	q.Add(update, preinstalled, install, remove, absent)

	return app.New().Run(ctx, q)
}
