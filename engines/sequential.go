package engines

import (
	"context"
	"fmt"

	"github.com/ntnn/tensile"
	"github.com/ntnn/tensile/facts"
	"golang.org/x/exp/slog"
)

type Sequential struct {
	Queue *tensile.Queue
	Facts facts.Facts
	log   *slog.Logger
}

func NewSequential(logger *slog.Logger) (*Sequential, error) {
	f, err := facts.New()
	if err != nil {
		return nil, fmt.Errorf("engines: error preparing facts: %w", err)
	}

	return &Sequential{
		Queue: tensile.NewQueue(f),
		Facts: f,
		log:   logger,
	}, nil
}

func (seq Sequential) Noop(ctx context.Context) error {
	return seq.run(ctx, false)
}

func (seq Sequential) Run(ctx context.Context) error {
	return seq.run(ctx, true)
}

func (seq Sequential) run(ctx context.Context, execute bool) error {
	seq.log.Info("getting facts")

	c, err := tensile.NewContext(
		ctx,
		nil,
		seq.Facts,
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

	seq.log.Info("channeling nodes from queue")
	ch := seq.Queue.Channel(ctx, isDone)

	for elem := range ch {
		ident := tensile.FormatIdentitier(elem)

		done[ident] = true

		log := seq.log.With(slog.String("node", ident))

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
