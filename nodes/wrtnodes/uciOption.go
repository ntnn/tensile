package wrtnodes

import (
	"fmt"

	"github.com/ntnn/tensile"
	"github.com/ntnn/tensile/nodes"
)

type UCIOptionState int

type UCIOption struct {
	// Config is the subsystem or config name.
	Config string

	// Name is the name of the section.
	Section string

	// Option is the name of the option.
	Option string

	// Value is the value to set.
	// Simple values are set.
	// TODO Lists are first removed and then set.
	//
	// If Value is nil the option is removed.
	Value any

	cmd nodes.Command
}

func (opt UCIOption) Shape() tensile.Shape {
	return UCI
}

func (opt UCIOption) Identifier() string {
	return fmt.Sprintf("%s.%s.%s", opt.Config, opt.Section, opt.Option)
}

func (opt *UCIOption) Validate() error {
	if opt.Config == "" {
		return fmt.Errorf("wrtnodes: config cannot be empty")
	}

	if opt.Section == "" {
		return fmt.Errorf("wrtnodes: section name cannot be empty")
	}

	if opt.Option == "" {
		return fmt.Errorf("wrtnodes: section type cannot be empty")
	}

	path := fmt.Sprintf(
		"%s.%s.%s",
		opt.Config,
		opt.Section,
		opt.Option,
	)

	if opt.Value == nil {
		opt.cmd = nodes.Command{
			Binary: "uci",
			Arguments: []string{
				"delete",
				path,
			},
		}
		return nil
	}

	// TODO handle lists
	opt.cmd = nodes.Command{
		Binary: "uci",
		Arguments: []string{
			"set",
			fmt.Sprintf("%s=%v", path, opt.Value),
		},
	}

	return nil
}

func (opt UCIOption) AfterNodes() []string {
	return []string{
		fmt.Sprintf("uci[%s.%s]", opt.Config, opt.Section),
	}
}

func (opt UCIOption) BeforeNodes() []string {
	return []string{
		"uci[commit]",
	}
}

func (opt UCIOption) Execute(ctx tensile.Context) (any, error) {
	return opt.cmd.Execute(ctx)
}
