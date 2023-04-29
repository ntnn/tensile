package nodes

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestCommand_Execute(t *testing.T) {
	sh := &Command{
		Description: "Hello, world!",
		Script:      "echo 'Hello, world!'",
	}
	require.Nil(t, sh.Validate())

	out, err := sh.Execute(Context(t))
	require.Nil(t, err)
	require.Equal(t, &CommandOutput{
		Result: 0,
		Stdout: "Hello, world!\n",
		Stderr: "",
	}, out)
}

func TestCommand_Execute_noscript(t *testing.T) {
	sh := &Command{
		Description: "Hello, world!",
		Arguments: []string{
			"-c",
			"echo 'Hello, world!'",
		},
	}
	require.Nil(t, sh.Validate())

	out, err := sh.Execute(Context(t))
	require.Nil(t, err)
	require.Equal(t, &CommandOutput{
		Result: 0,
		Stdout: "Hello, world!\n",
		Stderr: "",
	}, out)
}

func TestShell_Execute_input(t *testing.T) {
	sh := &Command{
		Description: "reading input",
		Script: `
while read input; do
	echo "read line: $input"
done
`,
		Input: []string{
			"first line",
			"second line",
			"third line",
		},
	}
	require.Nil(t, sh.Validate())

	out, err := sh.Execute(Context(t))
	require.Nil(t, err)
	require.Equal(t, &CommandOutput{
		Result: 0,
		Stdout: "read line: first line\nread line: second line\nread line: third line\n",
		Stderr: "",
	}, out)
}
