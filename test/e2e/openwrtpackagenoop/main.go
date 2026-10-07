// openwrtpackagenoop plans package changes without package lists.
package main

import (
	"context"
	"flag"
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
	unknown := flag.Bool("unknown", false, "add a package no repository provides")
	a := app.New()
	a.AddFlags(flag.CommandLine)
	flag.Parse()

	q := queue.New()

	update := &std.PackageManagerUpdate{}
	// not installed, unknown without package lists
	install := &std.Package{Name: "tree"}
	// not installed, must stay absent as a no-op
	absent := &std.Package{Name: "jq", State: tensile.Absent}

	q.Add(update, install, absent)

	if *unknown {
		q.Add(&std.Package{Name: "package-does-not-exist"})
	}

	return a.Run(ctx, q)
}
