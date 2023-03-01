package engines

import (
	"context"
)

type Simple struct {
	elements map[string]Identitier
	state    *simpleState
}

func NewSimple() *Simple {
	return &Simple{
		elements: map[string]Identitier{},
	}
}

func (simple *Simple) Add(elems ...Identitier) error {
	for _, elem := range elems {
		if validator, ok := elem.(Validator); ok {
			l.Debug("validating")
			if err := validator.Validate(); err != nil {
				return err
			}
		}

		if _, ok := simple.elements[elem.Identity()]; ok {
			return ErrSameIdentityAlreadyRegistered
		}
		simple.elements[elem.Identity()] = elem
	}
	return nil
}

func (simple Simple) Run(ctx context.Context) error {
	simple.state = newSimpleState()

	for {
		if len(simple.state.done) == len(simple.elements) {
			return nil
		}

		for _, elem := range simple.elements {
			// skip done elements
			if _, ok := simple.state.done[elem.Identity()]; ok {
				continue
			}

			if !simple.preElementsDone(elem) {
				continue
			}

			executor, ok := elem.(Executor)
			if !ok {
				simple.state.done[elem.Identity()] = true
				continue
			}

			if err := executor.Execute(ctx); err != nil {
				return err
			}

			simple.state.done[elem.Identity()] = true
		}
	}
}

func (simple Simple) hasElems(elems ...string) bool {
	for _, elem := range elems {
		if _, ok := simple.elements[elem]; !ok {
			return false
		}
	}
	return true
}

func (simple Simple) preElementsDone(elem any) bool {
	preElementer, ok := elem.(PreElementer)
	if !ok {
		return true
	}

	for _, preElement := range preElementer.PreElements() {
		if !simple.hasElems(preElement) {
			continue
		}

		done, ok := simple.state.done[preElement]
		if !ok {
			return false
		}

		if !done {
			return false
		}
	}

	return true
}
