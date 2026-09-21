package engine

import (
	"time"

	"github.com/ntnn/tensile"
)

// Stage is a phase of a single node execution.
type Stage string

// Stages of a single node execution.
const (
	StageValidate       Stage = "validate"
	StageNeedsExecution Stage = "needsExecution"
	StageExecute        Stage = "execute"
	StageReport         Stage = "report"
)

// Outcome is the result of a single node execution.
type Outcome string

const (
	// OutcomeExecuted marks nodes that reached Execute.
	OutcomeExecuted Outcome = "executed"
	// OutcomeSkipped marks nodes that reported no need for execution.
	OutcomeSkipped Outcome = "skipped"
	// OutcomeNoop marks nodes skipped by [Options.Noop].
	OutcomeNoop Outcome = "noop"
	// OutcomeFailed marks nodes that returned an error.
	OutcomeFailed Outcome = "failed"
)

// NodeSummary records outcome and per-stage timing of one node
// execution.
type NodeSummary struct {
	Identity tensile.Identity
	Outcome  Outcome
	Start    time.Time
	End      time.Time
	Stages   map[Stage]time.Duration
	Err      error
}

// Duration returns the total wall time of the node execution.
func (ns NodeSummary) Duration() time.Duration {
	return ns.End.Sub(ns.Start)
}

// Summary is the summary of a run.
type Summary struct {
	// Start is the timestamp when the run started.
	Start time.Time
	// End is the timestamp when the run finished.
	End time.Time
}
