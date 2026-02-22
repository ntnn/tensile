package queue

import (
	"fmt"
	"sync"

	"github.com/ntnn/tensile"
)

// Work is the result of building a queue.
type Work struct {
	providedValues map[string][]int64

	lock  sync.Mutex
	done  map[int64]struct{}
	order []*tensile.Node
}

// Get returns the next node that is ready to be executed.
// If there are no nodes ready to be executed, it returns nil and false.
// If all nodes are done, it returns nil and true.
func (w *Work) Get() (*tensile.Node, bool) {
	w.lock.Lock()
	defer w.lock.Unlock()

	if len(w.order) == 0 {
		// All nodes are done
		return nil, true
	}

	for _, node := range w.order {
		ready, err := w.isReady(node)
		if err != nil {
			// If there is an error checking if the node is ready, skip it
			continue
		}
		if ready {
			w.order = w.order[1:]
			return node, false
		}
	}
	return nil, false
}

func (w *Work) isReady(node *tensile.Node) (bool, error) {
	dependencies, err := node.DependsOn()
	if err != nil {
		return false, fmt.Errorf("failed to get dependencies for node with ID %d: %w", node.ID(), err)
	}

	for _, dep := range dependencies {
		providers, exists := w.providedValues[dep]
		if !exists {
			// The dependency is not provided by any node, skip
			continue
		}
		for _, providerId := range providers {
			if _, done := w.done[providerId]; !done {
				// The provider of the dependency is not done, so this node is not ready
				return false, nil
			}
		}
	}

	return true, nil
}

// MarkDone marks the given node as done. It should be called after a node has been executed.
func (w *Work) MarkDone(node *tensile.Node) {
	w.lock.Lock()
	defer w.lock.Unlock()
	w.done[node.ID()] = struct{}{}
}
