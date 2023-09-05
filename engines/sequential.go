package engines

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/ntnn/tensile"
)

type Sequential struct {
	Config *Config
}

func NewSequential(config *Config) (*Sequential, error) {
	if config == nil {
		var err error
		config, err = NewConfig()
		if err != nil {
			return nil, fmt.Errorf("engines: error in default config: %w", err)
		}
	}

	return &Sequential{
		Config: config,
	}, nil
}

func (seq Sequential) Noop(ctx context.Context) error {
	return seq.run(ctx, false)
}

func (seq Sequential) Run(ctx context.Context) error {
	return seq.run(ctx, true)
}

func (seq Sequential) run(ctx context.Context, execute bool) error {
	seq.Config.Log.Info("getting facts")

	c, err := tensile.NewContext(
		ctx,
		nil,
		seq.Config.Facts,
	)
	if err != nil {
		return fmt.Errorf("engines: error creating tensile.Context: %w", err)
	}

	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	seq.Config.Log.Info("channeling nodes from queue")
	ch, errCh := seq.Config.Queue.Channel(ctx)

	for nw := range ch {
		log := seq.Config.Log.With(slog.String("node", nw.String()))
		log.Debug("handling node")

		needsExecution, err := nw.NeedsExecution(c)
		if err != nil {
			return err
		}
		if !needsExecution {
			log.Debug("node does not need execution")
			continue
		}
		log.Debug("node needs execution")

		if !execute {
			log.Debug("would execute node")
			continue
		}

		// TODO handle result
		log.Debug("executing")
		if _, err := nw.Execute(c); err != nil {
			return err
		}
	}
	if err := <-errCh; err != nil {
		return err
	}

	return nil
}
