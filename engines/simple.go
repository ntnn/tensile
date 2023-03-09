package engines

import (
	"context"
	"fmt"

	"github.com/ntnn/tensile"
	"github.com/ntnn/tensile/facts"
	"golang.org/x/exp/slog"
)

type Simple struct {
	Queue *tensile.Queue
	log   *slog.Logger
}

func NewSimple(logger *slog.Logger) *Simple {
	return &Simple{
		Queue: tensile.NewQueue(),
		log:   logger,
	}
}

func (simple Simple) Run(ctx context.Context) error {
	f, err := facts.New()
	if err != nil {
		return fmt.Errorf("engines: error preparing facts: %w", err)
	}

	ctx = context.WithValue(ctx, facts.CtxFacts, f)

	done := map[string]bool{}
	isDone := func(idents ...string) bool {
		for _, ident := range idents {
			if _, ok := done[ident]; !ok {
				return false
			}
		}
		return true
	}

	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	ch := simple.Queue.Channel(ctx, isDone)

	for elem := range ch {
		ident := tensile.FormatIdentitier(elem)

		done[ident] = true

		executor, ok := elem.(tensile.Executor)
		if !ok {
			continue
		}

		if err := executor.Execute(ctx); err != nil {
			return err
		}
	}

	return nil
}
