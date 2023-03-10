package nodes

import (
	"github.com/ntnn/tensile"
	"golang.org/x/exp/slog"
)

// Log logs the given message on the given logger on the info level.
// If no logger is given the default slog logger is used.
type Log struct {
	Logger  *slog.Logger
	Message string
}

func (log Log) Validate() error {
	if log.Logger == nil {
		log.Logger = slog.Default()
	}
	return nil
}

func (log Log) Identity() (tensile.Shape, string) {
	return tensile.Noop, log.Message
}

func (log Log) Execute(ctx tensile.Context) error {
	log.Logger.Info(log.Message)
	return nil
}
