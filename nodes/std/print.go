package std

import (
	"fmt"
	"strings"

	"github.com/ntnn/tensile"
)

// Print prints a message when executed.
type Print struct {
	Message string
	Args    []any
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
