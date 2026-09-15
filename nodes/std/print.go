package std

import (
	"fmt"
	"strings"

	"github.com/ntnn/tensile"
)

var _ tensile.Identifier = (*Print)(nil)
var _ tensile.Executor = (*Print)(nil)

// Print prints a message when executed.
type Print struct {
	Message string
	Args    []any
}

// Identity implements [tensile.Identifier].
func (p *Print) Identity() tensile.Identity {
	return tensile.AsIdentity("print", "message", p.Message)
}

// NeedsExecution returns true, indicating the node should always execute.
func (p *Print) NeedsExecution(_ tensile.Wire) (bool, error) {
	return true, nil
}

// Execute implements [tensile.Executor].
func (p *Print) Execute(_ tensile.Wire) error {
	msg := p.Message
	if !strings.HasSuffix(msg, "\n") {
		msg += "\n"
	}
	fmt.Printf(msg, p.Args...)
	return nil
}
