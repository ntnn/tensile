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

	wire := &tensile.DefaultWire{
		Ctx: ctx,
		Log: opts.Logger.With("id", node.ID()),
	}

	if err := node.Validate(wire); err != nil {
		return fmt.Errorf("node validation failed: %w", err)
	}

	needsExecution, err := node.NeedsExecution(wire)
	if err != nil {
		return fmt.Errorf("failed to check if node with ID %d needs execution: %w", node.ID(), err)
	}
	if !needsExecution {
		opts.Logger.Debug(fmt.Sprintf("node with ID %d does not need execution, marking as done", node.ID()))
		work.MarkDone(node, false)
		return nil
	}

	if opts.Noop {
		opts.Logger.Debug(fmt.Sprintf("noop is enabled, skipping execution of node with ID %d", node.ID()))
		work.MarkDone(node, true)
		summary.IncrementNodesExecuted()
		return nil
	}

	if err := node.Execute(wire); err != nil {
		return fmt.Errorf("failed to execute node with ID %d: %w", node.ID(), err)
	}

	opts.Logger.Debug(fmt.Sprintf("successfully executed node with ID %d", node.ID()))
	work.MarkDone(node, true)
	summary.IncrementNodesExecuted()
	return nil
}
