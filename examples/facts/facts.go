package main

import (
	"context"
	"log"

	"github.com/ntnn/tensile"
	"github.com/ntnn/tensile/engines"
	"golang.org/x/exp/slog"
)

var _ tensile.Node = (*AccessFacts)(nil)

type AccessFacts struct {
}

func (af AccessFacts) Identity() (tensile.Shape, string) {
	return tensile.Noop, "log hostname from facts"
}

func (af AccessFacts) Validate() error {
	return nil
}

func (af AccessFacts) Execute(ctx tensile.Context) (any, error) {
	ctx.Logger(af).Info("hostname from facts",
		slog.String("hostname", ctx.Facts().Hostname),
	)
	return nil, nil
}

func main() {
	tensile.SetDebugLog()
	if err := doMain(); err != nil {
		log.Fatal(err)
	}
}

func doMain() error {
	seq, err := engines.NewSequential(slog.Default())
	if err != nil {
		return err
	}

	if err := seq.Queue.Add(&AccessFacts{}); err != nil {
		return err
	}

	return seq.Run(context.Background())
}
