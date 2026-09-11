package std

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var errPacmanNotFound = errors.New("exit status 1")

func pacmanFake(run func(ctx context.Context, args ...string) ([]byte, error)) *Pacman {
	return &Pacman{
		run:       run,
		available: func() bool { return true },
	}
}

// pacmanNotFound renders pacman output for an unknown package.
func pacmanNotFound(name string) []byte {
	return []byte("error: package '" + name + "' was not found\n")
}

// pacmanResult is a canned pacman invocation result.
type pacmanResult struct {
	out []byte
	err error
}

var (
	knownPkg   = pacmanResult{out: []byte("Version : 1.0-1\n")}
	unknownPkg = pacmanResult{out: pacmanNotFound("pkg"), err: errPacmanNotFound}
	failedRun  = pacmanResult{out: []byte("boom"), err: errors.New("run failed")}
)

func TestPacman_Handles(t *testing.T) {
	t.Parallel()

	cases := map[string]struct {
		want      bool
		wantErr   bool
		available bool
		si        pacmanResult
		qi        pacmanResult
	}{
		"in sync db":           {want: true, available: true, si: knownPkg},
		"local only":           {want: true, available: true, si: unknownPkg, qi: knownPkg},
		"unknown":              {available: true, si: unknownPkg, qi: unknownPkg},
		"pacman not available": {},
		"run failure":          {wantErr: true, available: true, si: failedRun},
	}

	for title, cas := range cases {
		t.Run(title, func(t *testing.T) {
			t.Parallel()

			pacman := &Pacman{
				available: func() bool { return cas.available },
				run: func(_ context.Context, args ...string) ([]byte, error) {
					result := cas.si
					if args[0] == "-Qi" {
						result = cas.qi
					}
					return result.out, result.err
				},
			}

			got, err := pacman.Handles(t.Context(), "pkg")
			if cas.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, cas.want, got)
		})
	}
}

func TestPacman_Installed(t *testing.T) {
	t.Parallel()

	errRun := errors.New("run failed")

	cases := map[string]struct {
		want    bool
		wantErr bool
		query   pacmanResult
	}{
		"installed": {
			want: true,
			query: pacmanResult{
				out: []byte("pkg 1.0-1\n"),
			},
		},
		"not installed": {query: unknownPkg},
		"run failure": {
			wantErr: true,
			query: pacmanResult{
				out: []byte("boom"),
				err: errRun,
			},
		},
	}

	for title, cas := range cases {
		t.Run(title, func(t *testing.T) {
			t.Parallel()

			pacman := pacmanFake(func(_ context.Context, _ ...string) ([]byte, error) {
				return cas.query.out, cas.query.err
			})

			got, err := pacman.Installed(t.Context(), "pkg")
			if cas.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, cas.want, got)
		})
	}
}

func TestPacman_Install(t *testing.T) {
	t.Parallel()

	var gotArgs []string
	pacman := pacmanFake(func(_ context.Context, args ...string) ([]byte, error) {
		gotArgs = args
		return nil, nil
	})

	require.NoError(t, pacman.Install(t.Context(), "pkg"))
	assert.Equal(t, []string{"-S", "--noconfirm", "--needed", "--", "pkg"}, gotArgs)
}

func TestPacman_Remove(t *testing.T) {
	t.Parallel()

	var gotArgs []string
	pacman := pacmanFake(func(_ context.Context, args ...string) ([]byte, error) {
		gotArgs = args
		return nil, nil
	})

	require.NoError(t, pacman.Remove(t.Context(), "pkg"))
	assert.Equal(t, []string{"-R", "--noconfirm", "--", "pkg"}, gotArgs)
}
