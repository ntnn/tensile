package std

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// shortTimeout keeps waits that cannot succeed short.
const shortTimeout = 10 * time.Millisecond

func TestWaitActive(t *testing.T) {
	t.Parallel()

	errStatus := errors.New("status failed")

	cases := map[string]struct {
		wantErr bool
		active  bool
		timeout time.Duration
		// statuses are returned in order, the last one repeats
		statuses []bool
		err      error
		// block makes status checks wait for the deadline
		block bool
	}{
		"already active":         {active: true, statuses: []bool{true}},
		"becomes active":         {active: true, statuses: []bool{false, true}},
		"becomes inactive":       {active: false, statuses: []bool{true, false}},
		"never active":           {wantErr: true, active: true, timeout: shortTimeout, statuses: []bool{false}},
		"status error":           {wantErr: true, active: true, err: errStatus},
		"deadline during status": {wantErr: true, active: true, timeout: shortTimeout, block: true},
	}

	for title, cas := range cases {
		t.Run(title, func(t *testing.T) {
			t.Parallel()

			calls := 0
			mgr := &fakeServiceManager{
				status: func(ctx context.Context, _ string) (ServiceStatus, error) {
					if cas.block {
						<-ctx.Done()
						return ServiceStatus{}, ctx.Err()
					}
					if cas.err != nil {
						return ServiceStatus{}, cas.err
					}
					active := cas.statuses[min(calls, len(cas.statuses)-1)]
					calls++
					return ServiceStatus{Active: active}, nil
				},
			}

			err := waitActive(t.Context(), mgr, "svc", cas.active, cas.timeout)
			if cas.wantErr {
				require.Error(t, err)
				if cas.timeout != 0 {
					require.ErrorContains(t, err, "waiting for service", "timeouts must report the wait")
				}
				return
			}
			require.NoError(t, err)
			assert.Equal(t, len(cas.statuses), calls, "must poll until the desired state is seen")
		})
	}
}
