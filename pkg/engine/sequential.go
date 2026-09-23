package engine

import (
	"context"
	"time"

	"github.com/ntnn/tensile/pkg/queue"
)

// Sequential is a simple execution engine that executes nodes in the
// order the work queue yields them without parallelisation.
// If any node errors the execution is stopped and the error is returned.
type Sequential struct {
	opts Options

	work    *queue.Work
	summary *Summary
	records []NodeSummary
}

// NewSequential creates a new Sequential execution engine.
func NewSequential(work *queue.Work, opts Options) *Sequential {
	s := new(Sequential)
	s.opts = opts.WithDefaults()
	s.work = work
	s.summary = &Summary{}
	return s
}

// Summary returns the summary of the current execution state.
func (s *Sequential) Summary() *Summary {
	return s.summary
}

// Records returns the per-node execution records.
func (s *Sequential) Records() []NodeSummary {
	return s.records
}

// Execute executes the nodes in the work queue.
func (s *Sequential) Execute(ctx context.Context) error {
	s.opts.Logger.Info("starting engine")

	s.summary.Start = time.Now()
	defer func() {
		s.summary.End = time.Now()
		s.summary.Analyze(s.records)
	}()

	for {
		s.opts.Logger.Debug("getting next node from work queue")
		node, done := s.work.Get()
		if done {
			s.opts.Logger.Info("all nodes are done, stopping")
			return nil
		}

		ns, err := executeNode(ctx, s.opts, s.work, node)
		s.records = append(s.records, ns)
		if err != nil {
			return err
		}
	}
}
