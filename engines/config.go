package engines

import (
	"fmt"
	"log/slog"

	"github.com/ntnn/tensile"
	"github.com/ntnn/tensile/facts"
)

// Common config for all engines.
type Config struct {
	Queue *tensile.Queue
	Facts facts.Facts
	Log   *slog.Logger
}

func NewConfig() (*Config, error) {
	c := new(Config)

	f, err := facts.New()
	if err != nil {
		return nil, fmt.Errorf("engines: error preparing facts: %w", err)
	}
	c.Facts = f

	c.Queue = tensile.NewQueue()
	c.Log = slog.Default()

	return c, nil
}
