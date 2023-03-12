package tensile

import (
	"context"
	"fmt"

	"github.com/ntnn/tensile/facts"
	"golang.org/x/exp/slog"
)

type Context interface {
	Context() context.Context
	Logger(Identitier) *slog.Logger
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

func (ec TContext) Logger(ident Identitier) *slog.Logger {
	return ec.logger.With(slog.String("identity", FormatIdentitier(ident)))
}

func (ec TContext) Result(shape Shape, name string) (any, bool, error) {
	return nil, false, fmt.Errorf("not implemented")
}

func (ec TContext) Facts() facts.Facts {
	return ec.facts
}
