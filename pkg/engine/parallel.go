package engine

import (
	"context"
	"runtime"
	"time"

	"github.com/ntnn/tensile/pkg/queue"
	"golang.org/x/sync/errgroup"
)

// ParallelOptions are options for the Parallel execution engine.
type ParallelOptions struct {
	Options

	// Workers is the number of worker goroutines executing nodes.
	// Defaults to [runtime.NumCPU].
	Workers int
}

// WithDefaults returns a ParallelOptions with default values.
func (o ParallelOptions) WithDefaults() ParallelOptions {
	o.Options = o.Options.WithDefaults()
	if o.Workers <= 0 {
		o.Workers = runtime.NumCPU()
	}
	return o
}

// Parallel is an execution engine that executes nodes in the order the
// work queue yields them with up to [ParallelOptions.Workers] nodes in
// parallel.
// If any node errors the execution of new nodes is stopped and the
// error is returned.
type Parallel struct {
	opts ParallelOptions

	work    *queue.Work
	summary *Summary
	records []NodeSummary
}

// NewParallel creates a new Parallel execution engine.
func NewParallel(work *queue.Work, opts ParallelOptions) *Parallel {
	p := new(Parallel)
	p.opts = opts.WithDefaults()
	p.work = work
	p.summary = &Summary{}
	return p
}

// Summary returns the summary of the current execution state.
func (p *Parallel) Summary() *Summary {
	return p.summary
}

// Records returns the per-node execution records.
// Only complete after Execute returned.
func (p *Parallel) Records() []NodeSummary {
	return p.records
}

// Execute executes the nodes in the work queue.
func (p *Parallel) Execute(ctx context.Context) error {
	p.opts.Logger.Info("starting engine", "workers", p.opts.Workers)

	p.summary.Start = time.Now()
	defer func() {
		p.summary.End = time.Now()
		p.summary.Analyze(p.records)
	}()

	g, ctx := errgroup.WithContext(ctx)
	items := p.work.Chan(ctx)
	locals := make([][]NodeSummary, p.opts.Workers)
	for i := range p.opts.Workers {
		g.Go(func() error {
			for item := range items {
				if item.Err != nil {
					return item.Err
				}
				ns, err := executeNode(ctx, p.opts.Options, p.work, item.Node)
				locals[i] = append(locals[i], ns)
				if err != nil {
					return err
				}
			}
			return nil
		})
	}
	err := g.Wait()
	for _, local := range locals {
		p.records = append(p.records, local...)
	}
	if err != nil {
		return err
	}

	p.opts.Logger.Info("all nodes are done, stopping")
	return nil
}
