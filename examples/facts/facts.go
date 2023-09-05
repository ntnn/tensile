package main

import (
	"context"
	"log"
	"log/slog"

	"github.com/ntnn/tensile"
	"github.com/ntnn/tensile/engines"
)

var _ tensile.Node = (*AccessFacts)(nil)

type AccessFacts struct {
}

func (af AccessFacts) Shape() tensile.Shape {
	return tensile.Noop
}

func (af AccessFacts) Identifier() string {
	return "log hostname from facts"
}

func (af AccessFacts) Validate() error {
	return nil
}

func (af AccessFacts) Execute(ctx tensile.Context) (any, error) {
	ctx.Logger().Info("hostname from facts",
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
	seq, err := engines.NewSequential(nil)
	if err != nil {
		return err
	}

	if err := seq.Config.Queue.Add(&AccessFacts{}); err != nil {
		return err
	}

	return seq.Run(context.Background())
}
