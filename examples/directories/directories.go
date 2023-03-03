package main

import (
	"context"
	"log"

	"github.com/ntnn/gorrect"
	"github.com/ntnn/gorrect/engines"
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
		&gorrect.Dir{
			Target: "/tmp",
		},
		&gorrect.Dir{
			Target: "/tmp/gorrect",
		},
		&gorrect.Dir{
			Target: "/tmp/gorrect/a",
		},
		&gorrect.Dir{
			Target: "/tmp/gorrect/b",
		},
		&gorrect.File{
			Target: "/tmp/gorrect/a/f",
		},
		&gorrect.File{
			Target: "/tmp/gorrect/b/f",
		},
	); err != nil {
		return err
	}

	return simple.Run(context.Background())
}
