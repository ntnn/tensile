package tensile

import (
	"context"
	"fmt"
)

type Queue struct {
	elements map[string]Identitier

	QueueChannelLength int
}

func NewQueue() *Queue {
	q := new(Queue)
	q.elements = map[string]Identitier{}
	q.QueueChannelLength = 100
	return q
}

func (queue *Queue) Add(elems ...Identitier) error {
	for _, elem := range elems {
		if err := queue.add(elem); err != nil {
			return err
		}
	}
	return nil
}

func (queue *Queue) add(elem Identitier) error {
	if validator, ok := elem.(Validator); ok {
		if err := validator.Validate(); err != nil {
			return err
		}
	}

	ident := FormatIdentitier(elem)
	if _, ok := queue.elements[ident]; ok {
		return fmt.Errorf("same identity already registered")
	}

	queue.elements[ident] = elem
	return nil
}

func (queue Queue) Channel(ctx context.Context, isDone func(idents ...string) bool) chan Identitier {
	ch := make(chan Identitier, queue.QueueChannelLength)

	go func() {
		defer close(ch)

		sent := map[string]bool{}

		for {
			if len(sent) == len(queue.elements) {
				return
			}

			for _, elem := range queue.elements {
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
						if _, ok := queue.elements[pre]; !ok {
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
