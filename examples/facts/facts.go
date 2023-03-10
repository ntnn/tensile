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
	simple := engines.NewSimple(slog.Default())

	if err := simple.Queue.Add(&AccessFacts{}); err != nil {
		return err
	}

	return simple.Run(context.Background())
}
