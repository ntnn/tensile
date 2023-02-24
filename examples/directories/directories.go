package main

import (
	"context"
	"log"

	"github.com/ntnn/gorrect"
	"github.com/ntnn/gorrect/engines"
)

func main() {
	if err := doMain(); err != nil {
		log.Fatal(err)
	}
}

func doMain() error {
	simple := engines.NewSimple()

	tmp, err := gorrect.NewDir("/tmp")
	if err != nil {
		return err
	}

	tmpGorrect, err := gorrect.NewDir("/tmp/gorrect")
	if err != nil {
		return err
	}

	tmpGorrectA, err := gorrect.NewDir("/tmp/gorrect/a")
	if err != nil {
		return err
	}

	tmpGorrectB, err := gorrect.NewDir("/tmp/gorrect/b")
	if err != nil {
		return err
	}

	tmpGorrectAF, err := gorrect.NewFile("/tmp/gorrect/a/f")
	if err != nil {
		return err
	}

	tmpGorrectBF, err := gorrect.NewFile("/tmp/gorrect/b/f")
	if err != nil {
		return err
	}

	if err := simple.Add(
		tmp,
		tmpGorrect,
		tmpGorrectA,
		tmpGorrectB,
		tmpGorrectAF,
		tmpGorrectBF,
	); err != nil {
		return err
	}

	return simple.Run(context.Background())
}
