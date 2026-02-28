package engine

import (
	"fmt"

	"github.com/ntnn/tensile"
	"github.com/ntnn/tensile/pkg/queue"
)

// Sequential is a simple execution engine that executes nodes in the
// order the work queue yields them without parallelisation. If any node
// errors the execution is stopped and the error is returned.
type Sequential struct {
	opts Options

	work    *queue.Work
	summary *Summary
}

// quential creates a new Sequential execution engine.
func NewSequential(work *queue.Work, opts Options) *Sequential {
	s := new(Sequential)
	s.opts = opts.WithDefaults()
	s.work = work
	s.summary = &Summary{}
	return s
}

func (s *Sequential) Summary() *Summary {
	return s.summary
}

// Execute executes the nodes in the work queue.
func (s *Sequential) Execute(ctx tensile.Context) error {
	s.opts.Logger.Info("starting engine")
	for {
		s.opts.Logger.Debug("getting next node from work queue")
		node, done := s.work.Get()
		if done {
			s.opts.Logger.Info("all nodes are done, stopping")
			return nil
		}

		if err := s.executeNode(ctx, node); err != nil {
			return err
		}
	}
}

func (s *Sequential) executeNode(ctx tensile.Context, node *tensile.Node) error {
	if err := node.Validate(ctx); err != nil {
		return fmt.Errorf("node validation failed: %w", err)
	}

	needsExecution, err := node.NeedsExecution(ctx)
	if err != nil {
		return fmt.Errorf("failed to check if node with ID %d needs execution: %w", node.ID(), err)
	}
	if !needsExecution {
		s.opts.Logger.Debug(fmt.Sprintf("node with ID %d does not need execution, marking as done", node.ID()))
		s.work.MarkDone(node)
		return nil
	}

	if s.opts.Noop {
		s.opts.Logger.Debug(fmt.Sprintf("noop is enabled, skipping execution of node with ID %d", node.ID()))
		s.work.MarkDone(node)
		s.summary.NodesExecuted++
		return nil
	}

	if err := node.Execute(ctx); err != nil {
		return fmt.Errorf("failed to execute node with ID %d: %w", node.ID(), err)
	}

	s.opts.Logger.Debug(fmt.Sprintf("successfully executed node with ID %d", node.ID()))
	s.work.MarkDone(node)
	s.summary.NodesExecuted++
	return nil
}
