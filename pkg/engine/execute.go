package engine

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"time"

	"github.com/ntnn/tensile"
	"github.com/ntnn/tensile/pkg/queue"
	"github.com/ntnn/tensile/pkg/storage"
)

// executeNode executes a single node via runNode and records outcome,
// error and total wall time in the returned NodeSummary.
func executeNode(ctx context.Context, opts Options, work *queue.Work, node *tensile.Node) (NodeSummary, error) {
	ns := NodeSummary{
		Identity: node.Identity(),
		Start:    time.Now(),
	}
	// TODO(ntnn): refactor
	err := runNode(ctx, opts, work, node, &ns)
	ns.End = time.Now()
	ns.Err = err
	if err != nil {
		ns.Outcome = OutcomeFailed
	}
	return ns, err
}

// runNode validates and executes a single node and marks it done in the
// work queue, filling the outcome into ns.
func runNode(ctx context.Context, opts Options, work *queue.Work, node *tensile.Node, ns *NodeSummary) error {
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	logger := opts.Logger.With("node", node.Identity())

	wire := &tensile.DefaultWire{
		Ctx: ctx,
		Log: logger,
		Store: storage.New[tensile.Identity](&scopedBackend{
			backend: opts.Backend,
			node:    node.Identity(),
			deps:    work.Dependencies(node.Identity()),
		}),
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
		// report before MarkDone so dependers only run with the output in place
		if err := report(wire, opts, node); err != nil {
			return err
		}
		work.MarkDone(node, false)
		ns.Outcome = OutcomeSkipped
		return nil
	}

	if opts.Noop {
		logger.Debug("noop is enabled, skipping execution")
		if err := report(wire, opts, node); err != nil {
			return err
		}
		work.MarkDone(node, true)
		ns.Outcome = OutcomeNoop
		return nil
	}

	if err := node.Execute(wire); err != nil {
		return fmt.Errorf("failed to execute node %s: %w", node.Identity(), err)
	}

	logger.Debug("successfully executed node")
	if err := report(wire, opts, node); err != nil {
		return err
	}
	work.MarkDone(node, true)
	ns.Outcome = OutcomeExecuted
	return nil
}

// report stores the node's reported output under its identity.
func report(wire tensile.Wire, opts Options, node *tensile.Node) error {
	output, ok, err := node.Report(wire)
	if err != nil {
		return fmt.Errorf("error getting node %q report: %w", node.Identity(), err)
	}
	if !ok {
		// no report
		return nil
	}
	if err := opts.Backend.Store(node.Identity(), output); err != nil {
		return fmt.Errorf("error storing node %q report: %w", node.Identity(), err)
	}
	return nil
}

// scopedBackend restricts reads to the declared dependencies of a node.
type scopedBackend struct {
	backend storage.Backend[tensile.Identity]
	node    tensile.Identity
	deps    []tensile.Identity
}

var _ storage.Backend[tensile.Identity] = (*scopedBackend)(nil)

// Load implements [storage.Backend].
func (s *scopedBackend) Load(key tensile.Identity) (any, error) {
	if !slices.Contains(s.deps, key) {
		return nil, fmt.Errorf("%s is not a dependency of %s", key, s.node)
	}
	return s.backend.Load(key)
}

// Store implements [storage.Backend]. Nodes cannot store outputs directly.
func (s *scopedBackend) Store(tensile.Identity, any) error {
	return errors.New("store is read-only")
}
