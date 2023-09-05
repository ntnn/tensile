package tensile

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/ntnn/tensile/facts"
)

// TODO context should be initialized by the engine, then passed to node wrappers to fill out with e.g. the correct logger
type Context interface {
	Context() context.Context
	Logger() *slog.Logger
	Result(Shape, string) (any, bool, error)
	Facts() facts.Facts
}

var _ Context = (*TContext)(nil)

type TContext struct {
	ctx    context.Context
	logger *slog.Logger
	facts  facts.Facts
}

func NewContext(ctx context.Context, logger *slog.Logger, facts facts.Facts) (*TContext, error) {
	c := new(TContext)

	c.ctx = ctx
	if c.ctx == nil {
		c.ctx = context.Background()
	}

	c.logger = logger
	if c.logger == nil {
		// TODO noop logger
		c.logger = slog.Default()
	}

	c.facts = facts

	return c, nil
}

func (ec TContext) Context() context.Context {
	return ec.ctx
}

func (ec TContext) Logger() *slog.Logger {
	return ec.logger
	// return ec.logger.With(slog.String("identity", FormatIdentitier(ident)))
}

func (ec TContext) Result(shape Shape, name string) (any, bool, error) {
	return nil, false, fmt.Errorf("not implemented")
}

func (ec TContext) Facts() facts.Facts {
	return ec.facts
}
