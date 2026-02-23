package tensilestd

import (
	"fmt"
	"strings"
)

type Print struct {
	Message string
	Args    []any
}

func (p *Print) NeedsExecution() (bool, error) {
	return true, nil
}

func (p *Print) Execute() error {
	msg := p.Message
	if !strings.HasSuffix(msg, "\n") {
		msg += "\n"
	}
	fmt.Printf(msg, p.Args...)
	return nil
}
