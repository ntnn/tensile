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

func (simple Simple) Noop(ctx context.Context) error {
	return simple.run(ctx, false)
}

func (simple Simple) Run(ctx context.Context) error {
	return simple.run(ctx, true)
}

func (simple Simple) run(ctx context.Context, execute bool) error {
	simple.log.Info("getting facts")
	f, err := facts.New()
	if err != nil {
		return fmt.Errorf("engines: error preparing facts: %w", err)
	}

	c, err := tensile.NewContext(
		ctx,
		nil,
		f,
	)
	if err != nil {
		return fmt.Errorf("engines: error creating tensile.Context: %w", err)
	}

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

	simple.log.Info("channeling nodes from queue")
	ch := simple.Queue.Channel(ctx, isDone)

	for elem := range ch {
		ident := tensile.FormatIdentitier(elem)

		done[ident] = true

		log := simple.log.With(slog.String("node", ident))

		log.Debug("handling node")

		if needsExecutioner, ok := elem.(tensile.NeedsExecutioner); ok {
			log.Debug("node is NeedsExecutioner, checking need")
			needsExecution, err := needsExecutioner.NeedsExecution(c)
			if err != nil {
				return err
			}
			if !needsExecution {
				log.Debug("node is NeedsExecutioner, no execution need")
				continue
			}
			log.Debug("node is NeedsExecutioner, needs execution")
		}

		executor, ok := elem.(tensile.Executor)
		if !ok {
			log.Warn("node is not Executor")
			continue
		}

		if !execute {
			log.Debug("would execute node")
			continue
		}

		// TODO handle result
		log.Debug("executing")
		if _, err := executor.Execute(c); err != nil {
			return err
		}
	}

	return nil
}
