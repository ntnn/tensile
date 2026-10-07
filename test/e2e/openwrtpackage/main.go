// openwrtpackage installs a package through the detected manager.
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

	// not installed, must be installed
	install := &std.Package{Name: "tree"}
	// not installed, must stay absent as a no-op
	absent := &std.Package{Name: "jq", State: tensile.Absent}

	q.Add(update, install, absent)

	return app.New().Run(ctx, q)
}
