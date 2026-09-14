package std

import (
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/ntnn/tensile"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCommand_Validate(t *testing.T) {
	t.Parallel()

	if _, err := exec.LookPath("sh"); err != nil {
		t.Skip("sh not available")
	}

	cases := map[string]struct {
		wantErr bool
		command Command
	}{
		"defaults":        {command: Command{Command: "true"}},
		"missing command": {wantErr: true, command: Command{}},
		"missing interpreter binary": {
			wantErr: true,
			command: Command{Shell: "no-such-interpreter", Args: []string{"-c"}, Command: "true"},
		},
	}

	for title, cas := range cases {
		t.Run(title, func(t *testing.T) {
			t.Parallel()

			err := cas.command.Validate(&tensile.DefaultWire{})
			if cas.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
		})
	}
}

func TestCommand_ValidateUnknownInterpreter(t *testing.T) {
	t.Parallel()

	if _, err := exec.LookPath("sh"); err != nil {
		t.Skip("sh not available")
	}

	command := Command{
		Shell:   "sh",
		Command: "true",
	}
	require.NoError(t, command.Validate(&tensile.DefaultWire{}), "known interpreter needs no Args")

	command.Shell = os.Args[0]
	require.Error(t, command.Validate(&tensile.DefaultWire{}), "unknown interpreter without Args must fail")

	command.Args = []string{"-c"}
	require.NoError(t, command.Validate(&tensile.DefaultWire{}), "explicit Args allow any interpreter")
}

func TestCommand_args(t *testing.T) {
	t.Parallel()

	cases := map[string]struct {
		want    []string
		wantErr bool
		command Command
	}{
		"default sh":    {want: []string{"-c"}, command: Command{}},
		"bash":          {want: []string{"-c"}, command: Command{Shell: "bash"}},
		"ruby":          {want: []string{"-e"}, command: Command{Shell: "ruby"}},
		"pwsh":          {want: []string{"-NoProfile", "-Command"}, command: Command{Shell: "pwsh"}},
		"cmd":           {want: []string{"/C"}, command: Command{Shell: "cmd"}},
		"override":      {want: []string{"-x"}, command: Command{Shell: "bash", Args: []string{"-x"}}},
		"unknown shell": {wantErr: true, command: Command{Shell: "custom"}},
	}

	for title, cas := range cases {
		t.Run(title, func(t *testing.T) {
			t.Parallel()

			got, err := cas.command.args()
			if cas.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, cas.want, got)
		})
	}
}

func TestCommand_NeedsExecution(t *testing.T) {
	t.Parallel()

	existing := filepath.Join(t.TempDir(), "existing")
	require.NoError(t, os.WriteFile(existing, nil, 0o600))
	missing := filepath.Join(t.TempDir(), "missing")

	errExit := errors.New("exit status 1")

	cases := map[string]struct {
		want    bool
		guarded bool
		runErr  error
		command Command
	}{
		"no guards":        {want: true, command: Command{Command: "true"}},
		"creates existing": {command: Command{Command: "true", Creates: existing}},
		"creates missing":  {want: true, command: Command{Command: "true", Creates: missing}},
		"removes existing": {want: true, command: Command{Command: "true", Removes: existing}},
		"removes missing":  {command: Command{Command: "true", Removes: missing}},
		"unless succeeds":  {guarded: true, command: Command{Command: "true", Unless: "check"}},
		"unless fails": {
			want: true, guarded: true, runErr: errExit,
			command: Command{Command: "true", Unless: "check"},
		},
		"onlyif succeeds":        {want: true, guarded: true, command: Command{Command: "true", OnlyIf: "check"}},
		"onlyif fails":           {guarded: true, runErr: errExit, command: Command{Command: "true", OnlyIf: "check"}},
		"creates beats removes":  {command: Command{Command: "true", Creates: existing, Removes: existing}},
		"creates missing+unless": {guarded: true, command: Command{Command: "true", Creates: missing, Unless: "check"}},
	}

	for title, cas := range cases {
		t.Run(title, func(t *testing.T) {
			t.Parallel()

			command := cas.command
			if cas.guarded {
				command.run = func(_ tensile.Wire, _ ...string) ([]byte, error) {
					return nil, cas.runErr
				}
			}

			got, err := command.NeedsExecution(&tensile.DefaultWire{})
			require.NoError(t, err)
			assert.Equal(t, cas.want, got)
		})
	}
}

func TestCommand_Execute(t *testing.T) {
	t.Parallel()

	t.Run("passes interpreter args and command", func(t *testing.T) {
		t.Parallel()

		var gotArgs []string
		command := Command{
			Shell:   "bash",
			Command: "echo hi",
			run: func(_ tensile.Wire, args ...string) ([]byte, error) {
				gotArgs = args
				return nil, nil
			},
		}

		require.NoError(t, command.Execute(&tensile.DefaultWire{}))
		assert.Equal(t, []string{"-c", "echo hi"}, gotArgs)
	})

	t.Run("returns error with output", func(t *testing.T) {
		t.Parallel()

		command := Command{
			Command: "false",
			run: func(_ tensile.Wire, _ ...string) ([]byte, error) {
				return []byte("boom\n"), errors.New("exit status 1")
			},
		}

		err := command.Execute(&tensile.DefaultWire{})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "boom")
	})
}

func TestCommand_ExecuteReal(t *testing.T) {
	t.Parallel()

	if _, err := exec.LookPath("sh"); err != nil {
		t.Skip("sh not available")
	}

	target := filepath.Join(t.TempDir(), "made-by-command")
	command := Command{
		Command: "touch \"$TARGET\"",
		Env: map[string]string{
			"TARGET": target,
		},
		Creates: target,
	}

	wire := &tensile.DefaultWire{}
	require.NoError(t, command.Validate(wire))

	needs, err := command.NeedsExecution(wire)
	require.NoError(t, err)
	require.True(t, needs, "target missing, must need execution")

	require.NoError(t, command.Execute(wire))
	assert.FileExists(t, target)

	needs, err = command.NeedsExecution(wire)
	require.NoError(t, err)
	assert.False(t, needs, "target exists, creates guard must skip")
}
