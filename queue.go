package tensile

import (
	"context"
	"errors"
	"fmt"

	"github.com/ntnn/tensile/facts"
)

type Queue struct {
	nodes map[string]Identitier
	facts facts.Facts

	QueueChannelLength int
}

func NewQueue(facts facts.Facts) *Queue {
	q := new(Queue)
	q.nodes = map[string]Identitier{}
	q.facts = facts
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
	if _, ok := queue.nodes[ident]; ok {
		return fmt.Errorf("same identity already registered")
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
	ch, errCh, err := generator.Nodes(queue.facts)
	if err != nil {
		return fmt.Errorf("direct error from generator: %w", err)
	}

	errs := []error{}
	for node := range ch {
		if err := queue.add(node); err != nil {
			errs = append(errs, err)
		}
	}

	for err := range errCh {
		errs = append(errs, err)
	}

	return errors.Join(errs...)
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
