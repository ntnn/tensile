package queue

import (
	"context"
	"slices"
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
	// dependencies maps node identities to the identities of the nodes they depend on
	dependencies map[tensile.Identity][]tensile.Identity
	// handlers maps handler identities to the identities of nodes notifying them
	handlers map[tensile.Identity][]tensile.Identity

	lock sync.RWMutex
	cond *sync.Cond
	// done maps node identities to whether the node was executed.
	done  map[tensile.Identity]bool
	order []*tensile.Node

	// hels maps serialization keys to the identities of nodes that are
	// currently being executed
	held map[string]tensile.Identity
}

// newWork returns a Work with initialized internals.
func newWork() *Work {
	work := new(Work)
	work.cond = sync.NewCond(&work.lock)
	work.done = make(map[tensile.Identity]bool)
	work.dependencies = make(map[tensile.Identity][]tensile.Identity)
	work.held = make(map[string]tensile.Identity)
	return work
}

// Dependencies returns the identities of the direct dependencies of
// the node with the given identity.
func (w *Work) Dependencies(identity tensile.Identity) []tensile.Identity {
	return slices.Clone(w.dependencies[identity])
}

// Get returns the next node that is ready to be executed.
// If there are no nodes ready to be executed, it returns nil and false.
// If all nodes are done, it returns nil and true.
func (w *Work) Get() (*tensile.Node, bool) {
	w.lock.Lock()
	defer w.lock.Unlock()

	node := w.get()
	if node == nil {
		return nil, true
	}
	return node, false
}

// get returns the next ready node or nil if none is ready.
// The caller must hold the lock.
func (w *Work) get() *tensile.Node {
	for i := 0; i < len(w.order); {
		node := w.order[i]

		if !w.isReady(node) {
			i++
			continue
		}

		if w.isHeld(node) {
			i++
			continue
		}

		w.order = append(w.order[:i], w.order[i+1:]...)

		if node.Identity().Kind() == barrierKind {
			// Barriers only order the graph and are completed silently.
			// An end barrier has all of its groups nodes as
			// a dependency and may be used as a notifier for a handler,
			// so set true if any of its nodes was executed.
			w.done[node.Identity()] = w.wasNotified(node)
			continue
		}

		if w.isHandler(node) && !w.wasNotified(node) {
			// No notifying node was executed, skip the handler.
			w.done[node.Identity()] = false
			continue
		}

		for _, key := range node.SerializesOn() {
			if key == "" {
				continue
			}
			w.held[key] = node.Identity()
		}
		return node
	}

	return nil
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

		node := w.get()
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
	_, ok := w.handlers[node.Identity()]
	return ok
}

// wasNotified reports whether at least one notifier of the given handler node was executed.
func (w *Work) wasNotified(node *tensile.Node) bool {
	for _, notifier := range w.handlers[node.Identity()] {
		if w.done[notifier] {
			return true
		}
	}
	return false
}

// isReady reports whether all dependencies of the node are done.
func (w *Work) isReady(node *tensile.Node) bool {
	for _, dep := range w.dependencies[node.Identity()] {
		if _, done := w.done[dep]; !done {
			return false
		}
	}
	return true
}

// isHeld reports whether any serialization key is held by another node.
func (w *Work) isHeld(node *tensile.Node) bool {
	for _, key := range node.SerializesOn() {
		if _, held := w.held[key]; held {
			return true
		}
	}
	return false
}

// MarkDone marks the given node as done and records whether it was executed.
// It should be called after a node has been handled.
func (w *Work) MarkDone(node *tensile.Node, executed bool) {
	w.lock.Lock()
	defer w.lock.Unlock()
	w.done[node.Identity()] = executed
	for _, key := range node.SerializesOn() {
		delete(w.held, key)
	}
	w.cond.Broadcast()
}

// Executed returns whether the given node was executed and whether it is done.
func (w *Work) Executed(node *tensile.Node) (bool, bool) {
	w.lock.RLock()
	defer w.lock.RUnlock()
	executed, done := w.done[node.Identity()]
	return executed, done
}
