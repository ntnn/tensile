package tensile

import (
	"context"
	"errors"
	"fmt"
)

type Queue struct {
	nodes map[string]NodeWrapper
	// in which order nodes were added
	order []string

	QueueChannelLength int
}

func NewQueue() *Queue {
	q := new(Queue)
	q.nodes = map[string]NodeWrapper{}
	q.QueueChannelLength = 100
	return q
}

func (queue *Queue) Add(nodes ...Node) error {
	for _, node := range nodes {
		nw := NodeWrap(node)
		if err := queue.add(nw); err != nil {
			return err
		}
	}
	return nil
}

// NodeGenerator can return more nodes for a Queue to collect.
//
// This is primarily useful to have a single Noop node that dynamically
// generates more nodes based on configuration.
type NodeGenerator interface {
	Nodes() ([]Node, error)
}

func (queue *Queue) add(nw NodeWrapper) error {
	if err := nw.Validate(); err != nil {
		return err
	}

	if existing, ok := queue.nodes[nw.String()]; ok {
		if err := isCollisionBoth(existing, nw); err != nil {
			return fmt.Errorf("same identity already registered, collision check: %w", err)
		}
		return nil
	}

	queue.nodes[nw.String()] = nw
	queue.order = append(queue.order, nw.String())

	if generator, ok := nw.Node.(NodeGenerator); ok {
		if err := queue.addFrom(generator); err != nil {
			return err
		}
	}

	return nil
}

func (queue Queue) addFrom(generator NodeGenerator) error {
	nodes, err := generator.Nodes()
	if err != nil {
		return fmt.Errorf("direct error from generator: %w", err)
	}

	for _, node := range nodes {
		if err := queue.add(NodeWrap(node)); err != nil {
			return fmt.Errorf("error adding node %q: %w", node, err)
		}
	}

	return nil
}

var (
	ErrIsCollisionerNotImplemented = fmt.Errorf("nodes do not implement IsCollisioner interface")
)

func isCollisionBoth(a, b NodeWrapper) error {
	if err := a.IsCollision(b); !errors.Is(err, ErrIsCollisionerNotImplemented) {
		return err
	}

	if err := b.IsCollision(a); !errors.Is(err, ErrIsCollisionerNotImplemented) {
		return err
	}

	return ErrIsCollisionerNotImplemented
}

var (
	ErrCyclicalDependencies = fmt.Errorf("tensile: reached iteration limit, nodes have cyclical dependencies")
)

func (queue Queue) Channel(ctx context.Context) (chan NodeWrapper, chan error) {
	ch := make(chan NodeWrapper, queue.QueueChannelLength)
	errCh := make(chan error, 1)

	go func() {
		defer close(ch)
		defer close(errCh)

		// edges lists relations between nodes:
		//   edges[b][a]=true
		// where a is an earlier node in the execution.
		edges := map[string]map[string]bool{}

		for key, node := range queue.nodes {
			if _, ok := edges[key]; !ok {
				edges[key] = map[string]bool{}
			}
			for _, after := range node.AfterNodes() {
				if _, ok := queue.nodes[after]; !ok {
					// node does not exist, continue
					continue
				}
				edges[key][after] = true
			}

			for _, before := range node.BeforeNodes() {
				if _, ok := queue.nodes[before]; !ok {
					continue
				}
				if _, ok := edges[before]; !ok {
					edges[before] = map[string]bool{}
				}
				edges[before][key] = true
			}
		}

		sent := map[string]bool{}
		iterations := 0

	outer:
		for len(sent) < len(queue.order) {
			for _, key := range queue.order {
				if _, ok := sent[key]; ok {
					continue
				}

				dependcies, ok := edges[key]
				if !ok || len(dependcies) == 0 {
					iterations = 0
					ch <- queue.nodes[key]
					sent[key] = true
					// drop dependency from other nodes:w
					for b := range edges {
						delete(edges[b], key)
					}
					// continue with outer loop to ensure that nodes are passed along as they were queued
					continue outer
				}
			}

			iterations += 1
			if iterations > 3 {
				errCh <- ErrCyclicalDependencies
				return
			}
		}

	}()

	return ch, errCh
}
