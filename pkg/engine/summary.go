package engine

import (
	"sync/atomic"
	"time"
)

// Summary is the summary of a run.
type Summary struct {
	// Start is the timestamp when the run started.
	Start time.Time
	// End is the timestamp when the run finished.
	End time.Time

	// NodesExecuted is the number of executed nodes.
	NodesExecuted atomic.Int64
}
