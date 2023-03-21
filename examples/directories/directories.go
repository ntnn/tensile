package main

import (
	"context"
	"log"

	"github.com/ntnn/tensile/engines"
	"github.com/ntnn/tensile/nodes"

	// set debug logging
	_ "github.com/ntnn/tensile/testutils"
)

func main() {
	if err := doMain(); err != nil {
		log.Fatal(err)
	}
}

func doMain() error {
	seq, err := engines.NewSequential(nil)
	if err != nil {
		return err
	}

	if err := seq.Config.Queue.Add(
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

	return seq.Noop(context.Background())
}
