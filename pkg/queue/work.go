package queue

import (
	"fmt"
	"sync"

	"github.com/ntnn/tensile"
)

// Work is the result of building a queue.
type Work struct {
	providedRefs map[tensile.NodeRef][]int64
	// handlers maps handler node IDs to the IDs of nodes notifying them
	handlers map[int64][]int64

	lock sync.RWMutex
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

	if len(w.order) == 0 {
		// All nodes are done
		return nil, true, nil
	}

	node, err := w.takeReady()
	if err != nil {
		return nil, false, err
	}
	return node, false, nil
}

// takeReady removes and returns the next ready node from the order.
// Ready handler nodes whose notifying nodes were not executed are
// marked done and skipped.
// Returns nil if no node is ready.
func (w *Work) takeReady() (*tensile.Node, error) {
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
	return nil, nil
}

// isHandler returns true if the node has any notifiers.
func (w *Work) isHandler(node *tensile.Node) bool {
	return len(w.handlers[node.ID()]) > 0
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
}

// Executed returns whether the given node was executed and whether it is done.
func (w *Work) Executed(node *tensile.Node) (bool, bool) {
	w.lock.RLock()
	defer w.lock.RUnlock()
	executed, done := w.done[node.ID()]
	return executed, done
}
