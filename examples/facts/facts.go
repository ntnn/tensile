package main

import (
	"context"
	"fmt"
	"log"

	"github.com/ntnn/tensile"
	"github.com/ntnn/tensile/engines"
	"github.com/ntnn/tensile/facts"
	"golang.org/x/exp/slog"
)

type AccessFacts struct {
}

func (af AccessFacts) Identity() (tensile.Shape, string) {
	return tensile.Noop, ""
}

func (af AccessFacts) Execute(ctx context.Context) error {
	f, ok := ctx.Value(facts.CtxFacts).(*facts.Facts)
	if !ok {
		return fmt.Errorf("unable to retrieve facts from context")
	}

	log.Printf("value of hostname in facts: %q", f.Hostname)
	return nil
}

func main() {
	if err := doMain(); err != nil {
		log.Fatal(err)
	}
}

func doMain() error {
	simple := engines.NewSimple(slog.Default())

	if err := simple.Queue.Add(&AccessFacts{}); err != nil {
		return err
	}

	return simple.Noop(context.Background())
}
