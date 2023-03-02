package gorrect

import (
	"context"

	"golang.org/x/exp/slog"
)

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

func (log Log) Identity() (Identity, string) {
	return Noop, log.Message
}

func (log Log) Execute(ctx context.Context) error {
	log.Logger.Info(log.Message)
	return nil
}
