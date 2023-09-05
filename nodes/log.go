package nodes

import (
	"log/slog"

	"github.com/ntnn/tensile"
)

var _ tensile.Node = (*Log)(nil)

// Log logs the given message on the given logger on the info level.
// If no logger is given the default slog logger is used.
type Log struct {
	Logger  *slog.Logger
	Message string
}

func (log *Log) Validate() error {
	if log.Logger == nil {
		log.Logger = slog.Default()
	}
	return nil
}

func (log Log) Shape() tensile.Shape {
	return tensile.Noop
}

func (log Log) Identifier() string {
	return log.Message
}

func (log Log) Execute(ctx tensile.Context) (any, error) {
	log.Logger.Info(log.Message)
	return nil, nil
}
