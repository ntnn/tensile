package main

import (
	"context"
	"log"

	"github.com/ntnn/tensile"
	"github.com/ntnn/tensile/engines"
	"golang.org/x/exp/slog"
)

func main() {
	if err := doMain(); err != nil {
		log.Fatal(err)
	}
}

func doMain() error {
	simple := engines.NewSimple(slog.Default())

	if err := simple.Queue.Add(
		&tensile.Dir{
			Target: "/tmp",
		},
		&tensile.Dir{
			Target: "/tmp/tensile",
		},
		&tensile.Dir{
			Target: "/tmp/tensile/a",
		},
		&tensile.Dir{
			Target: "/tmp/tensile/b",
		},
		&tensile.File{
			Target: "/tmp/tensile/a/f",
		},
		&tensile.File{
			Target: "/tmp/tensile/b/f",
		},
	); err != nil {
		return err
	}

	return simple.Run(context.Background())
}
