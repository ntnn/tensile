package engine

import (
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/ntnn/tensile"
)

// Executor executes a single node once.
// An instance must not be reused.
type Executor struct {
	Logger *slog.Logger
	Noop   bool
	Node   *tensile.Node
	Wire   tensile.Wire

	// Store persists the node's reported output.
	Store func(tensile.Identity, any) error

	// Done marks the node done, changed reports whether it executed.
	Done func(changed bool)

	summary        NodeSummary
	needsExecution bool
}

// Run executes the node through its stages and returns the record.
func (e *Executor) Run() (NodeSummary, error) {
	if e.Logger == nil || e.Node == nil || e.Wire == nil || e.Store == nil || e.Done == nil {
		return NodeSummary{}, errors.New("executor requires Logger, Node, Wire, Store and Done")
	}

	e.summary = NodeSummary{
		Start:  time.Now(),
		Stages: map[Stage]time.Duration{},
	}
	err := e.run()
	e.summary.End = time.Now()
	e.summary.Err = err
	if err != nil {
		e.summary.Outcome = OutcomeFailed
	}
	return e.summary, err
}

func (e *Executor) run() error {
	e.summary.Identity = e.Node.Identity()

	if err := e.summary.stage(StageValidate, func() error {
		return e.Node.Validate(e.Wire)
	}); err != nil {
		return fmt.Errorf("node validation failed: %w", err)
	}

	if err := e.summary.stage(StageNeedsExecution, func() error {
		var err error
		e.needsExecution, err = e.Node.NeedsExecution(e.Wire)
		//nolint:wrapcheck // wrapped by caller
		return err
	}); err != nil {
		return fmt.Errorf("failed to check if node %s needs execution: %w", e.Node.Identity(), err)
	}

	if !e.needsExecution {
		e.Logger.Debug("node does not need execution, marking as done")
		return e.finish(OutcomeSkipped, false)
	}

	if e.Noop {
		e.Logger.Debug("noop is enabled, skipping execution")
		return e.finish(OutcomeNoop, true)
	}

	if err := e.summary.stage(StageExecute, func() error {
		return e.Node.Execute(e.Wire)
	}); err != nil {
		return fmt.Errorf("failed to execute node %s: %w", e.Node.Identity(), err)
	}

	e.Logger.Debug("successfully executed node")
	return e.finish(OutcomeExecuted, true)
}

func (e *Executor) finish(outcome Outcome, changed bool) error {
	if err := e.summary.stage(StageReport, e.report); err != nil {
		return err
	}
	e.Done(changed)
	e.summary.Outcome = outcome
	return nil
}

func (e *Executor) report() error {
	output, ok, err := e.Node.Report(e.Wire)
	if err != nil {
		return fmt.Errorf("error getting node %q report: %w", e.Node.Identity(), err)
	}
	if !ok {
		// no report
		return nil
	}
	if err := e.Store(e.Node.Identity(), output); err != nil {
		return fmt.Errorf("error storing node %q report: %w", e.Node.Identity(), err)
	}
	return nil
}
