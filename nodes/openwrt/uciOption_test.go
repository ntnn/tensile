package openwrt

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// fakeOption answers uci show with out/err and records staging verbs.
type fakeOption struct {
	out   string
	err   error
	calls [][]string
}

func (f *fakeOption) run(_ context.Context, args ...string) ([]byte, error) {
	if args[0] == "show" {
		return []byte(f.out), f.err
	}
	f.calls = append(f.calls, args)
	return nil, nil
}

func TestUCIOption_Validate(t *testing.T) {
	t.Parallel()

	cases := map[string]struct {
		option  UCIOption[string]
		wantErr bool
	}{
		"valid": {
			option: UCIOption[string]{Config: "c", Section: "s", Option: "o"},
		},
		"valid absent": {
			option: UCIOption[string]{Config: "c", Section: "s", Option: "o", State: UCIAbsent},
		},
		"missing config": {
			option:  UCIOption[string]{Section: "s", Option: "o"},
			wantErr: true,
		},
		"missing section": {
			option:  UCIOption[string]{Config: "c", Option: "o"},
			wantErr: true,
		},
		"missing option": {
			option:  UCIOption[string]{Config: "c", Section: "s"},
			wantErr: true,
		},
		"unknown state": {
			option:  UCIOption[string]{Config: "c", Section: "s", Option: "o", State: "bogus"},
			wantErr: true,
		},
	}

	for title, cas := range cases {
		t.Run(title, func(t *testing.T) {
			t.Parallel()

			err := cas.option.Validate(nil)
			if cas.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
		})
	}
}

func TestUCIOption_NeedsExecution_string(t *testing.T) {
	t.Parallel()

	errRun := errors.New("run failed")

	cases := map[string]struct {
		want    bool
		wantErr bool
		value   string
		state   UCIState
		out     string
		err     error
	}{
		"matches": {
			value: "world",
			out:   "t.main.hello='world'\n",
		},
		"differs": {
			want:  true,
			value: "world",
			out:   "t.main.hello='other'\n",
		},
		"missing": {
			want:  true,
			value: "world",
			out:   "uci: Entry not found",
			err:   errRun,
		},
		"absent but exists": {
			want:  true,
			state: UCIAbsent,
			out:   "t.main.hello='world'\n",
		},
		"absent and missing": {
			state: UCIAbsent,
			out:   "uci: Entry not found",
			err:   errRun,
		},
		"run failure": {
			wantErr: true,
			out:     "boom",
			err:     errRun,
		},
	}

	for title, cas := range cases {
		t.Run(title, func(t *testing.T) {
			t.Parallel()

			fake := &fakeOption{out: cas.out, err: cas.err}
			option := &UCIOption[string]{
				Config: "t", Section: "main", Option: "hello",
				Value: cas.value,
				State: cas.state,
				run:   fake.run,
			}

			got, _, err := option.NeedsExecution(testWire(t))
			if cas.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, cas.want, got)
		})
	}
}

func TestUCIOption_NeedsExecution_list(t *testing.T) {
	t.Parallel()

	cases := map[string]struct {
		want  bool
		value []string
		out   string
	}{
		"matches": {
			value: []string{"192.168.178.5", "1.1.1.1"},
			out:   "network.lan.dns='192.168.178.5' '1.1.1.1'\n",
		},
		"order differs": {
			want:  true,
			value: []string{"1.1.1.1", "192.168.178.5"},
			out:   "network.lan.dns='192.168.178.5' '1.1.1.1'\n",
		},
		"value differs": {
			want:  true,
			value: []string{"192.168.178.5"},
			out:   "network.lan.dns='192.168.178.5' '1.1.1.1'\n",
		},
		"quoted value with space": {
			value: []string{"managed-config", "other config"},
			out:   "dhcp.lan.ra_flags='managed-config' 'other config'\n",
		},
	}

	for title, cas := range cases {
		t.Run(title, func(t *testing.T) {
			t.Parallel()

			fake := &fakeOption{out: cas.out}
			option := &UCIOption[[]string]{
				Config: "c", Section: "s", Option: "o",
				Value: cas.value,
				run:   fake.run,
			}

			got, _, err := option.NeedsExecution(testWire(t))
			require.NoError(t, err)
			assert.Equal(t, cas.want, got)
		})
	}
}

func TestUCIOption_NeedsExecution_int(t *testing.T) {
	t.Parallel()

	fake := &fakeOption{out: "t.main.port='8080'\n"}
	option := &UCIOption[int]{
		Config: "t", Section: "main", Option: "port",
		Value: 8080,
		run:   fake.run,
	}

	got, _, err := option.NeedsExecution(testWire(t))
	require.NoError(t, err)
	assert.False(t, got)
}

func TestUCIOption_NeedsExecution_bool(t *testing.T) {
	t.Parallel()

	fake := &fakeOption{out: "t.main.ignore='1'\n"}
	option := &UCIOption[bool]{
		Config: "t", Section: "main", Option: "ignore",
		Value: true,
		run:   fake.run,
	}

	got, _, err := option.NeedsExecution(testWire(t))
	require.NoError(t, err)
	assert.False(t, got)
}

func TestUCIOption_NeedsExecution_intList(t *testing.T) {
	t.Parallel()

	fake := &fakeOption{out: "t.main.ports='80' '443'\n"}
	option := &UCIOption[[]int]{
		Config: "t", Section: "main", Option: "ports",
		Value: []int{80, 443},
		run:   fake.run,
	}

	got, _, err := option.NeedsExecution(testWire(t))
	require.NoError(t, err)
	assert.False(t, got)
}

func TestUCIOption_Execute_scalar(t *testing.T) {
	t.Parallel()

	fake := &fakeOption{}
	option := &UCIOption[string]{
		Config: "t", Section: "main", Option: "hello",
		Value: "world",
		run:   fake.run,
	}

	_, err := option.Execute(testWire(t))
	require.NoError(t, err)
	assert.Equal(t, [][]string{{"set", "t.main.hello=world"}}, fake.calls)
}

func TestUCIOption_Execute_list(t *testing.T) {
	t.Parallel()

	fake := &fakeOption{}
	option := &UCIOption[[]string]{
		Config: "network", Section: "lan", Option: "dns",
		Value: []string{"192.168.178.5", "1.1.1.1"},
		run:   fake.run,
	}

	_, err := option.Execute(testWire(t))
	require.NoError(t, err)
	assert.Equal(t, [][]string{
		{"delete", "network.lan.dns"},
		{"add_list", "network.lan.dns=192.168.178.5"},
		{"add_list", "network.lan.dns=1.1.1.1"},
	}, fake.calls, "lists are rebuilt in declared order")
}

func TestUCIOption_Execute_absent(t *testing.T) {
	t.Parallel()

	fake := &fakeOption{}
	option := &UCIOption[string]{
		Config: "t", Section: "main", Option: "legacy",
		State: UCIAbsent,
		run:   fake.run,
	}

	_, err := option.Execute(testWire(t))
	require.NoError(t, err)
	assert.Equal(t, [][]string{{"delete", "t.main.legacy"}}, fake.calls)
}

func TestUCIOption_Execute_deleteMissingIsNoop(t *testing.T) {
	t.Parallel()

	errRun := errors.New("exit status 1")
	option := &UCIOption[string]{
		Config: "t", Section: "main", Option: "legacy",
		State: UCIAbsent,
		run: func(_ context.Context, _ ...string) ([]byte, error) {
			return []byte("uci: Entry not found\n"), errRun
		},
	}

	_, err := option.Execute(testWire(t))
	require.NoError(t, err)
}
