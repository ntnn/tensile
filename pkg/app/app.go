package app

import (
	"context"
	"flag"
	"fmt"
	"io"
	"os"

	"github.com/ntnn/tensile/pkg/engine"
	"github.com/ntnn/tensile/pkg/queue"
)

// App wraps the defaults shared by tensile binaries.

// App is a basic wrapper for a tensile binary executing [queue.Queue] with [engine.Parallel].
type App struct {
	// Parallel configures the Parallel execution engine.
	Parallel engine.ParallelOptions
	// Render configures the summary rendering.
	Render engine.RenderOptions
	// Log configures the logger.
	Log LogOptions
	// Out receives the rendered summary.
	Out io.Writer
}

// New returns an App writing to [os.Stdout].
func New() *App {
	return &App{
		Out: os.Stdout,
	}
}

// AddFlags binds the flag-configurable options to fs.
func (a *App) AddFlags(fs *flag.FlagSet) {
	a.Parallel.AddFlags(fs)
	a.Render.AddFlags(fs)
	a.Log.AddFlags(fs)
}

// Run builds q, executes it with the Parallel engine and renders the summary to Out.
// The summary is rendered even when execution fails.
func (a *App) Run(ctx context.Context, q *queue.Queue) error {
	if a.Parallel.Logger == nil {
		logger, err := a.Log.Logger(os.Stderr)
		if err != nil {
			return err
		}
		a.Parallel.Logger = logger
	}

	work, err := q.Build()
	if err != nil {
		return fmt.Errorf("building work queue: %w", err)
	}

	eng := engine.NewParallel(work, a.Parallel)
	execErr := eng.Execute(ctx)

	if err := eng.Summary().Render(a.Out, a.Render); err != nil {
		return err
	}
	return execErr
}
