package wrtnodes

import (
	"fmt"

	"github.com/ntnn/tensile"
	"github.com/ntnn/tensile/nodes"
)

type UCISection struct {
	// Config is the subsystem or config name.
	Config string

	// Name is the name of the section.
	// TODO can/should empty names be allowed?
	Name string

	// Type of the section
	Type string

	cmd nodes.Command
}

func (section UCISection) Shape() tensile.Shape {
	return UCI
}

func (section UCISection) Identifier() string {
	return fmt.Sprintf("%s.%s", section.Config, section.Name)
}

func (section *UCISection) Validate() error {
	if section.Config == "" {
		return fmt.Errorf("wrtnodes: config cannot be empty")
	}

	if section.Name == "" {
		return fmt.Errorf("wrtnodes: section name cannot be empty")
	}

	if section.Type == "" {
		return fmt.Errorf("wrtnodes: section type cannot be empty")
	}

	section.cmd = nodes.Command{
		Binary: "uci",
		Arguments: []string{
			"set",
			fmt.Sprintf(
				"%s.%s=%s",
				section.Config,
				section.Name,
				section.Type,
			),
		},
	}

	return nil
}

func (section UCISection) BeforeNodes() []string {
	return []string{
		"uci[commit]",
	}
}

func (section UCISection) Execute(ctx tensile.Context) (any, error) {
	return section.cmd.Execute(ctx)
}
