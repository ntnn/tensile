package main

import (
	"context"
	"log"

	"github.com/ntnn/tensile"
	"github.com/ntnn/tensile/engines"
	"github.com/ntnn/tensile/nodes"
	"golang.org/x/exp/slog"
)

func main() {
	tensile.SetDebugLog()
	if err := doMain(); err != nil {
		log.Fatal(err)
	}
}

func doMain() error {
	simple := engines.NewSimple(slog.Default())

	if err := simple.Queue.Add(
		&nodes.Dir{
			Target: "/tmp",
		},
		&nodes.Dir{
			Target: "/tmp/tensile",
		},
		&nodes.Dir{
			Target: "/tmp/tensile/a",
		},
		&nodes.Dir{
			Target: "/tmp/tensile/b",
		},
		&nodes.File{
			Target: "/tmp/tensile/a/f",
		},
		&nodes.File{
			Target: "/tmp/tensile/b/f",
		},
	); err != nil {
		return err
	}

	return simple.Noop(context.Background())
}
