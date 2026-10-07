package engine

import (
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/ntnn/tensile"
	"github.com/ntnn/tensile/pkg/diff"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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
				record("pkg", OutcomeSatisfied, 40*time.Millisecond, 0),
			},
			wantNodes: 4,
			wantByOutcome: map[Outcome]int{
				OutcomeExecuted:  2,
				OutcomeFailed:    1,
				OutcomeSatisfied: 1,
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

func TestSummary_Render(t *testing.T) {
	t.Parallel()

	start := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	summary := Summary{
		Start: start,
		End:   start.Add(time.Second),
	}
	summary.Analyze([]NodeSummary{
		{
			Identity: tensile.AsIdentity("iniField", "path", "/etc/b.conf", "key", "x"),
			Outcome:  OutcomeExecuted,
			Diff: diff.NewFieldChanges(&diff.FieldChange{
				Field: "x",
				Old:   diff.Absent,
				New:   "1",
			}),
		},
		{
			Identity: tensile.AsIdentity("package", "name", "satisfied"),
			Outcome:  OutcomeSatisfied,
		},
		{
			Identity: tensile.AsIdentity("chown", "path", "/etc/b.conf"),
			Outcome:  OutcomeExecuted,
		},
		{
			Identity: tensile.AsIdentity("serviceRestart", "name", "a"),
			Outcome:  OutcomeHandlerExecuted,
		},
		{
			Identity: tensile.AsIdentity("file", "path", "/etc/a.conf"),
			Outcome:  OutcomeFailed,
			Err:      errors.New("boom"),
		},
		{
			Identity: tensile.AsIdentity("serviceRestart", "name", "b"),
			Outcome:  OutcomeNotNotified,
		},
	})

	var out strings.Builder
	require.NoError(t, summary.Render(&out, RenderOptions{}))
	assert.Equal(t, `run: 1s, 6 nodes, satisfied: 1, executed: 2, failed: 1, handler not notified: 1, handler executed: 1
/etc/a.conf
  file: failed: boom
/etc/b.conf
  x: (absent) -> 1
  chown: changed
serviceRestart[name="a"]
  changed
`, out.String())
}
