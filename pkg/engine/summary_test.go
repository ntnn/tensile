package engine

import (
	"testing"
	"time"

	"github.com/ntnn/tensile"
	"github.com/stretchr/testify/assert"
)

func TestSummary_Analyze(t *testing.T) {
	t.Parallel()

	record := func(kind string, outcome Outcome, validate, execute time.Duration) NodeSummary {
		return NodeSummary{
			Identity: tensile.AsIdentity(kind),
			Outcome:  outcome,
			Stages: map[Stage]time.Duration{
				StageValidate: validate,
				StageExecute:  execute,
			},
		}
	}

	cases := map[string]struct {
		records            []NodeSummary
		wantNodes          int
		wantByOutcome      map[Outcome]int
		wantStageTotals    map[Stage]time.Duration
		wantStageAvgByKind map[string]map[Stage]time.Duration
	}{
		"empty input": {
			records:            nil,
			wantNodes:          0,
			wantByOutcome:      map[Outcome]int{},
			wantStageTotals:    map[Stage]time.Duration{},
			wantStageAvgByKind: map[string]map[Stage]time.Duration{},
		},
		"single record": {
			records: []NodeSummary{
				record("file", OutcomeExecuted, 10*time.Millisecond, 100*time.Millisecond),
			},
			wantNodes: 1,
			wantByOutcome: map[Outcome]int{
				OutcomeExecuted: 1,
			},
			wantStageTotals: map[Stage]time.Duration{
				StageValidate: 10 * time.Millisecond,
				StageExecute:  100 * time.Millisecond,
			},
			wantStageAvgByKind: map[string]map[Stage]time.Duration{
				"file": {
					StageValidate: 10 * time.Millisecond,
					StageExecute:  100 * time.Millisecond,
				},
			},
		},
		"averages per kind, totals over all": {
			records: []NodeSummary{
				record("file", OutcomeExecuted, 10*time.Millisecond, 100*time.Millisecond),
				record("file", OutcomeExecuted, 30*time.Millisecond, 300*time.Millisecond),
				record("file", OutcomeFailed, 20*time.Millisecond, 200*time.Millisecond),
				record("pkg", OutcomeSkipped, 40*time.Millisecond, 0),
			},
			wantNodes: 4,
			wantByOutcome: map[Outcome]int{
				OutcomeExecuted: 2,
				OutcomeFailed:   1,
				OutcomeSkipped:  1,
			},
			wantStageTotals: map[Stage]time.Duration{
				StageValidate: 100 * time.Millisecond,
				StageExecute:  600 * time.Millisecond,
			},
			wantStageAvgByKind: map[string]map[Stage]time.Duration{
				"file": {
					StageValidate: 20 * time.Millisecond,
					StageExecute:  200 * time.Millisecond,
				},
				"pkg": {
					StageValidate: 40 * time.Millisecond,
					StageExecute:  0,
				},
			},
		},
	}

	for title, cas := range cases {
		t.Run(title, func(t *testing.T) {
			t.Parallel()

			var summary Summary
			summary.Analyze(cas.records)

			assert.Equal(t, cas.wantNodes, summary.Nodes)
			assert.Equal(t, cas.wantByOutcome, summary.ByOutcome)
			assert.Equal(t, cas.wantStageTotals, summary.StageTotals)
			assert.Equal(t, cas.wantStageAvgByKind, summary.StageAvgByKind)
		})
	}
}
