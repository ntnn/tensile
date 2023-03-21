package tensile

import (
	"context"
	"fmt"
)

type Queue struct {
	nodes map[string]Identitier

	QueueChannelLength int
}

func NewQueue() *Queue {
	q := new(Queue)
	q.nodes = map[string]Identitier{}
	q.QueueChannelLength = 100
	return q
}

func (queue *Queue) Add(nodes ...Identitier) error {
	for _, node := range nodes {
		if err := queue.add(node); err != nil {
			return err
		}
	}
	return nil
}

func (queue *Queue) add(node Identitier) error {
	if validator, ok := node.(Validator); ok {
		if err := validator.Validate(); err != nil {
			return err
		}
	}

	ident := FormatIdentitier(node)
	if existing, ok := queue.nodes[ident]; ok {
		if err := isCollisionBoth(existing, node); err != nil {
			return fmt.Errorf("same identity already registered, collision check: %w", err)
		}
		return nil
	}

	queue.nodes[ident] = node

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

func isCollisionBoth(a, b Identitier) error {
	if err := isCollision(a, b); err != ErrIsCollisionerNotImplemented {
		return err
	}

	if err := isCollision(b, a); err != ErrIsCollisionerNotImplemented {
		return err
	}

	return ErrIsCollisionerNotImplemented
}

func isCollision(a, b Identitier) error {
	isCollisioner, ok := a.(IsCollisioner)
	if !ok {
		return ErrIsCollisionerNotImplemented
	}

	return isCollisioner.IsCollision(b)
}

func (queue Queue) Channel(ctx context.Context, isDone func(idents ...string) bool) chan Identitier {
	ch := make(chan Identitier, queue.QueueChannelLength)

	go func() {
		defer close(ch)

		sent := map[string]bool{}

		for {
			if len(sent) == len(queue.nodes) {
				return
			}

			for _, elem := range queue.nodes {
				if err := ctx.Err(); err != nil {
					return
				}

				ident := FormatIdentitier(elem)

				if _, ok := sent[ident]; ok {
					continue
				}

				if preElementer, ok := elem.(PreElementer); ok {
					// collect which pre elements need to be checked
					checkPres := []string{}
					for _, pre := range preElementer.PreElements() {
						// filter elements that do not exist in the
						// queue
						if _, ok := queue.nodes[pre]; !ok {
							continue
						}
						checkPres = append(checkPres, pre)
					}

					// check pre elements
					if !isDone(checkPres...) {
						continue
					}
				}

				ch <- elem
				sent[ident] = true
			}
		}
	}()

	return ch
}
