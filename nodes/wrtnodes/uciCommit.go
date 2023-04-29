package wrtnodes

import (
	"fmt"

	"github.com/ntnn/tensile"
	"github.com/ntnn/tensile/nodes"
)

type UCICommit struct {
}

func (c UCICommit) Shape() tensile.Shape {
	return UCI
}

func (c UCICommit) Identifier() string {
	return "commit"
}

func (c *UCICommit) Validate() error {
	return nil
}

type UCICommitOutput struct {
	Result  int
	Changes string
}

func (c UCICommit) Execute(ctx tensile.Context) (any, error) {
	changesCmd := &nodes.Command{
		Binary:    "uci",
		Arguments: []string{"changes"},
	}
	if err := changesCmd.Validate(); err != nil {
		return nil, err
	}

	changesOutRaw, err := changesCmd.Execute(ctx)
	if err != nil {
		return nil, fmt.Errorf("nodeswrt: error getting changes: %w", err)
	}
	changesOut := changesOutRaw.(*nodes.CommandOutput)

	out := &UCICommitOutput{
		Changes: changesOut.Stdout,
	}

	commitCmd := &nodes.Command{
		Binary:    "uci",
		Arguments: []string{"commit"},
	}
	if err := commitCmd.Validate(); err != nil {
		return nil, err
	}

	commitOutRaw, err := commitCmd.Execute(ctx)
	if err != nil {
		return nil, fmt.Errorf("nodeswrt: error comitting changes: %w", err)
	}
	commitOut := commitOutRaw.(*nodes.CommandOutput)

	out.Result = commitOut.Result

	return out, nil
}
