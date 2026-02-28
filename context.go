package tensile

import "context"

// Context is a superset of [context.Context] to pass additional fields to a [Node].
type Context interface {
	context.Context
}

// TensileCtx implements a [Context].
type TensileCtx struct { //nolint:revive
	context.Context //nolint:containedctx

	// TODO remove TensileCtx, it's not a good idea
}

// NewContext returns an initialized [TensileCtx].
func NewContext(ctx context.Context) *TensileCtx {
	return &TensileCtx{ctx}
}
