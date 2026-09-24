package engine

import (
	"fmt"
	"log/slog"
	"maps"
	"slices"
	"strings"
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

// stageOrder is the execution order of stages for rendering.
var stageOrder = []Stage{
	StageValidate,
	StageNeedsExecution,
	StageExecute,
	StageReport,
}

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
	Diff     tensile.Diff
	Err      error
}

// Duration returns the total wall time of the node execution.
func (ns NodeSummary) Duration() time.Duration {
	return ns.End.Sub(ns.Start)
}

// stage runs fn and adds its duration to Stages under st.
func (ns *NodeSummary) stage(st Stage, fn func() error) error {
	start := time.Now()
	err := fn()
	ns.Stages[st] += time.Since(start)
	return err
}

// Summary is the computed result of a run over its node records.
type Summary struct {
	// Start is the timestamp when the run started.
	Start time.Time
	// End is the timestamp when the run finished.
	End time.Time

	// Records are the records most of the values are computed from.
	Records []NodeSummary

	// Nodes is the number of node records.
	Nodes int
	// ByOutcome counts nodes per outcome.
	ByOutcome map[Outcome]int
	// StageAvgByKind is the average stage duration per node kind.
	StageAvgByKind map[string]map[Stage]time.Duration
	// StageTotals is the total time spent per stage.
	StageTotals map[Stage]time.Duration
}

// Duration returns the total wall time of the node execution.
func (s *Summary) Duration() time.Duration {
	return s.End.Sub(s.Start)
}

// Analyze processes [NodeSummary] to get insights into the overall run.
// Repeated calls of Analyze will overwrite previous results.
func (s *Summary) Analyze(records []NodeSummary) {
	s.Records = records
	s.Nodes = len(records)
	s.ByOutcome = map[Outcome]int{}
	s.StageAvgByKind = map[string]map[Stage]time.Duration{}
	s.StageTotals = map[Stage]time.Duration{}

	counts := map[string]map[Stage]int{}
	for _, record := range records {
		s.ByOutcome[record.Outcome]++

		kind := record.Identity.Kind()
		if s.StageAvgByKind[kind] == nil {
			s.StageAvgByKind[kind] = map[Stage]time.Duration{}
			counts[kind] = map[Stage]int{}
		}
		for stage, duration := range record.Stages {
			s.StageTotals[stage] += duration
			s.StageAvgByKind[kind][stage] += duration
			counts[kind][stage]++
		}
	}

	for kind, stages := range s.StageAvgByKind {
		for stage := range stages {
			stages[stage] /= time.Duration(counts[kind][stage])
		}
	}
}

// String implements [fmt.Stringer].
// It renders human-readable multi-line output.
func (s Summary) String() string {
	var b strings.Builder

	fmt.Fprintf(&b, "run: %s, %d nodes\n", s.Duration(), s.Nodes)

	b.WriteString("outcomes:\n")
	for _, outcome := range slices.Sorted(maps.Keys(s.ByOutcome)) {
		fmt.Fprintf(&b, "  %s: %d\n", outcome, s.ByOutcome[outcome])
	}

	b.WriteString("stage totals:\n")
	for _, stage := range stageOrder {
		fmt.Fprintf(&b, "  %s: %s\n", stage, s.StageTotals[stage])
	}

	b.WriteString("stage averages per kind:\n")
	for _, kind := range slices.Sorted(maps.Keys(s.StageAvgByKind)) {
		fmt.Fprintf(&b, "  %s:\n", kind)
		stages := s.StageAvgByKind[kind]
		for _, stage := range stageOrder {
			fmt.Fprintf(&b, "    %s: %s\n", stage, stages[stage])
		}
	}

	diffs := false
	for _, record := range s.Records {
		if record.Diff == nil {
			continue
		}
		if !diffs {
			b.WriteString("diffs:\n")
			diffs = true
		}
		fmt.Fprintf(&b, "  %s:\n", record.Identity)
		for line := range strings.Lines(record.Diff.String()) {
			b.WriteString("    ")
			b.WriteString(strings.TrimSuffix(line, "\n"))
			b.WriteByte('\n')
		}
	}

	return strings.TrimSuffix(b.String(), "\n")
}

// LogValue implements [slog.LogValuer].
func (s Summary) LogValue() slog.Value {
	outcomes := make([]slog.Attr, 0, len(s.ByOutcome))
	for _, outcome := range slices.Sorted(maps.Keys(s.ByOutcome)) {
		outcomes = append(outcomes, slog.Int(string(outcome), s.ByOutcome[outcome]))
	}

	totals := make([]slog.Attr, 0, len(stageOrder))
	for _, stage := range stageOrder {
		totals = append(totals, slog.Duration(string(stage), s.StageTotals[stage]))
	}

	avgs := make([]slog.Attr, 0, len(s.StageAvgByKind))
	for _, kind := range slices.Sorted(maps.Keys(s.StageAvgByKind)) {
		stages := s.StageAvgByKind[kind]
		attrs := make([]slog.Attr, 0, len(stageOrder))
		for _, stage := range stageOrder {
			attrs = append(attrs, slog.Duration(string(stage), stages[stage]))
		}
		avgs = append(avgs, slog.Attr{Key: kind, Value: slog.GroupValue(attrs...)})
	}

	return slog.GroupValue(
		slog.Time("start", s.Start),
		slog.Time("end", s.End),
		slog.Duration("duration", s.Duration()),
		slog.Int("nodes", s.Nodes),
		slog.Attr{Key: "outcomes", Value: slog.GroupValue(outcomes...)},
		slog.Attr{Key: "stageTotals", Value: slog.GroupValue(totals...)},
		slog.Attr{Key: "stageAvgByKind", Value: slog.GroupValue(avgs...)},
	)
}
