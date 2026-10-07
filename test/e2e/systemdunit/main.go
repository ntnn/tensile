// systemdunit deploys a unit with a drop-in and prunes unmanaged drop-ins.
package main

import (
	"context"
	"flag"
	"log"

	"github.com/coreos/go-systemd/v22/unit"
	"github.com/ntnn/tensile"
	"github.com/ntnn/tensile/nodes/std"
	"github.com/ntnn/tensile/nodes/systemd"
	"github.com/ntnn/tensile/pkg/app"
	"github.com/ntnn/tensile/pkg/queue"
)

const name = "e2e-unit.service"

func main() {
	if err := run(context.Background()); err != nil {
		log.Fatal(err)
	}
}

func run(ctx context.Context) error {
	description := flag.String("description", "tensile e2e drop-in", "description set by the drop-in")
	a := app.New()
	a.AddFlags(flag.CommandLine)
	flag.Parse()

	q := queue.New()

	base := &systemd.Unit{
		Name: name,
		Sections: []*unit.UnitSection{
			{
				Section: "Unit",
				Entries: []*unit.UnitEntry{
					{Name: "Description", Value: "tensile e2e unit"},
				},
			},
			{
				Section: "Service",
				Entries: []*unit.UnitEntry{
					{Name: "ExecStart", Value: "/bin/sleep infinity"},
				},
			},
		},
	}
	dropIn := &systemd.Unit{
		Name:   name,
		DropIn: "10-e2e",
		Sections: []*unit.UnitSection{
			{
				Section: "Unit",
				Entries: []*unit.UnitEntry{
					{Name: "Description", Value: *description},
				},
			},
		},
	}
	prune := &systemd.PruneDropIns{Name: name}
	reload := tensile.NewHandler(&systemd.DaemonReload{})

	service := &std.Service{
		Name:    name,
		Running: new(true),
	}

	q.Add(base, dropIn, prune, reload, service)
	q.DependsOn(service, base, dropIn, prune)

	return a.Run(ctx, q)
}
