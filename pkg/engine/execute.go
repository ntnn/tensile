package engine

import (
	"context"

	"github.com/ntnn/tensile"
	"github.com/ntnn/tensile/pkg/queue"
	"github.com/ntnn/tensile/pkg/storage"
)

func executeNode(ctx context.Context, opts Options, work *queue.Work, node *tensile.Node) (NodeSummary, error) {
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	// TODO(ntnn): further dissolve this, engines should be able to use
	// Executor directly with a few seams.

	logger := opts.Logger.With("node", node.Identity())

	executor := &Executor{
		Logger: logger,
		Noop:   opts.Noop,
		Node:   node,
		Wire: &tensile.DefaultWire{
			Ctx: ctx,
			Log: logger,
			Store: storage.New[tensile.Identity](&scopedBackend{
				backend: opts.Backend,
				node:    node.Identity(),
				deps:    work.Dependencies(node.Identity()),
			}),
			Topo: work,
		},
		Store: opts.Backend.Store,
		Done: func(changed bool) {
			work.MarkDone(node, changed)
		},
	}
	return executor.Run()
}

func notNotifiedRecords(work *queue.Work) []NodeSummary {
	dropped := work.NotNotified()
	records := make([]NodeSummary, 0, len(dropped))
	for _, node := range dropped {
		records = append(records, NodeSummary{
			Identity: node.Identity(),
			Outcome:  OutcomeNotNotified,
		})
	}
	return records
}
