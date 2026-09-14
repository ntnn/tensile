package queue

import (
	"context"
	"fmt"
	"sync"

	"github.com/ntnn/tensile"
)

// Item is a single element yielded by [Work.Chan].
// Exactly one field is set.
type Item struct {
	Node *tensile.Node
	Err  error
}

// Work is the result of building a queue.
type Work struct {
	providedRefs map[tensile.NodeRef][]int64
	// handlers maps handler node IDs to the IDs of nodes notifying them
	handlers map[int64][]int64

	lock sync.RWMutex
	cond *sync.Cond
	// done maps node IDs to whether the node was executed.
	done  map[int64]bool
	order []*tensile.Node
}

// Get returns the next node that is ready to be executed.
// If there are no nodes ready to be executed, it returns nil and false.
// If all nodes are done, it returns nil and true.
func (w *Work) Get() (*tensile.Node, bool, error) {
	w.lock.Lock()
	defer w.lock.Unlock()

	node, err := w.get()
	if err != nil {
		return nil, false, err
	}
	if node == nil {
		return nil, true, nil
	}
	return node, false, nil
}

// get returns the next ready node or nil if none is ready.
// The caller must hold the lock.
func (w *Work) get() (*tensile.Node, error) {
	for i := 0; i < len(w.order); {
		node := w.order[i]

		ready, err := w.isReady(node)
		if err != nil {
			return nil, err
		}

		if !ready {
			i++
			continue
		}

		w.order = append(w.order[:i], w.order[i+1:]...)

		if w.isHandler(node) && !w.wasNotified(node) {
			// No notifying node was executed, skip the handler.
			w.done[node.ID()] = false
			continue
		}

		return node, nil
	}

	return nil, nil //nolint:nilnil // nil node means none ready
}

// Chan returns a channel yielding nodes that are ready to be executed.
// The channel is closed once all nodes have been processed.
func (w *Work) Chan(ctx context.Context) <-chan Item {
	items := make(chan Item)

	// Wake up all goroutines waiting on the condition if the context is cancelled.
	// That should be only the [Work.next] goroutine, which then exits
	// due to the cancelled context.
	stop := context.AfterFunc(ctx, w.cond.Broadcast)

	go func() {
		defer close(items)
		defer stop()

		for {
			node, err := w.next(ctx)
			if err != nil {
				items <- Item{Err: err}
				return
			}
			if node == nil {
				return
			}

			select {
			case items <- Item{Node: node}:
			case <-ctx.Done():
				items <- Item{Err: ctx.Err()}
				return
			}
		}
	}()

	return items
}

func (w *Work) next(ctx context.Context) (*tensile.Node, error) {
	w.lock.Lock()
	defer w.lock.Unlock()

	for {
		if err := ctx.Err(); err != nil {
			return nil, err
		}

		node, err := w.get()
		if err != nil {
			return nil, err
		}
		if node != nil {
			return node, nil
		}
		if len(w.order) == 0 {
			return nil, nil //nolint:nilnil // all nodes yielded
		}

		// No new node, wait on the condition which unlocks the lock and
		// wait for a broadcast, which happens once [Work.MarkDone]
		// marks a node as done.
		w.cond.Wait()
	}
}

// isHandler returns true if the node is registered as a handler.
func (w *Work) isHandler(node *tensile.Node) bool {
	_, ok := w.handlers[node.ID()]
	return ok
}

// wasNotified reports whether at least one notifier of the given handler node was executed.
func (w *Work) wasNotified(node *tensile.Node) bool {
	for _, notifierID := range w.handlers[node.ID()] {
		if w.done[notifierID] {
			return true
		}
	}
	return false
}

func (w *Work) isReady(node *tensile.Node) (bool, error) {
	// Handlers are only ready once all their notifiers are done.
	for _, notifierID := range w.handlers[node.ID()] {
		if _, done := w.done[notifierID]; !done {
			return false, nil
		}
	}

	dependencies, err := node.DependsOn()
	if err != nil {
		return false, fmt.Errorf("failed to get dependencies for node with ID %d: %w", node.ID(), err)
	}

	for _, dep := range dependencies {
		providers, exists := w.providedRefs[dep]
		if !exists {
			// The dependency is not provided by any node, skip
			continue
		}
		for _, providerID := range providers {
			if _, done := w.done[providerID]; !done {
				// The provider of the dependency is not done, so this node is not ready
				return false, nil
			}
		}
	}

	return true, nil
}

// MarkDone marks the given node as done and records whether it was executed.
// It should be called after a node has been handled.
func (w *Work) MarkDone(node *tensile.Node, executed bool) {
	w.lock.Lock()
	defer w.lock.Unlock()
	w.done[node.ID()] = executed
	w.cond.Broadcast()
}

// Executed returns whether the given node was executed and whether it is done.
func (w *Work) Executed(node *tensile.Node) (bool, bool) {
	w.lock.RLock()
	defer w.lock.RUnlock()
	executed, done := w.done[node.ID()]
	return executed, done
}
