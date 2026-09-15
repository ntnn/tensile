package engine

import (
	"context"
	"fmt"

	"github.com/ntnn/tensile"
	"github.com/ntnn/tensile/pkg/queue"
)

// executeNode validates and executes a single node and marks it done in the work queue.
func executeNode(ctx context.Context, opts Options, work *queue.Work, summary *Summary, node *tensile.Node) error {
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	logger := opts.Logger.With("node", node.Identity())

	wire := &tensile.DefaultWire{
		Ctx: ctx,
		Log: logger,
	}

	if err := node.Validate(wire); err != nil {
		return fmt.Errorf("node validation failed: %w", err)
	}

	needsExecution, err := node.NeedsExecution(wire)
	if err != nil {
		return fmt.Errorf("failed to check if node %s needs execution: %w", node.Identity(), err)
	}
	if !needsExecution {
		logger.Debug("node does not need execution, marking as done")
		work.MarkDone(node, false)
		return nil
	}

	if opts.Noop {
		logger.Debug("noop is enabled, skipping execution")
		work.MarkDone(node, true)
		summary.IncrementNodesExecuted()
		return nil
	}

	if err := node.Execute(wire); err != nil {
		return fmt.Errorf("failed to execute node %s: %w", node.Identity(), err)
	}

	logger.Debug("successfully executed node")
	work.MarkDone(node, true)
	summary.IncrementNodesExecuted()
	return nil
}
