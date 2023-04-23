package tensile

import (
	"context"
	"errors"
	"fmt"
)

type Queue struct {
	nodes map[string]NodeWrapper

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
		if err := queue.add(node); err != nil {
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

func (queue *Queue) add(node Node) error {
	nw := NodeWrap(node)

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

	if generator, ok := node.(NodeGenerator); ok {
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
		if err := queue.add(node); err != nil {
			return fmt.Errorf("error adding node %q: %w", node, err)
		}
	}

	return nil
}

var (
	ErrIsCollisionerNotImplemented = fmt.Errorf("nodes do not implement IsCollisioner interface")
)

func isCollisionBoth(a, b NodeWrapper) error {
	if err := a.IsCollision(b); err != nil && !errors.Is(err, ErrIsCollisionerNotImplemented) {
		return err
	}

	if err := b.IsCollision(a); err != nil && !errors.Is(err, ErrIsCollisionerNotImplemented) {
		return err
	}

	return nil
}

func (queue Queue) Channel(ctx context.Context, isDone func(idents ...string) bool) chan NodeWrapper {
	ch := make(chan NodeWrapper, queue.QueueChannelLength)

	go func() {
		defer close(ch)

		sent := map[string]bool{}

		for {
			if len(sent) == len(queue.nodes) {
				return
			}

			for _, nw := range queue.nodes {
				if err := ctx.Err(); err != nil {
					return
				}

				if _, ok := sent[nw.String()]; ok {
					continue
				}

				// collect which pre elements need to be checked
				checkPres := []string{}
				for _, before := range nw.BeforeNodes() {
					// filter elements that do not exist in the
					// queue
					if _, ok := queue.nodes[before]; !ok {
						continue
					}
					checkPres = append(checkPres, before)
				}

				ch <- nw
				sent[nw.String()] = true
			}
		}
	}()

	return ch
}
