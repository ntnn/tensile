package tensile

import "context"

// Context is a superset of [context.Context] to pass additional fields to a [Node].
type Context interface {
	context.Context
}

type TensileCtx struct {
	context.Context
}

func NewContext(ctx context.Context) *TensileCtx {
	return &TensileCtx{ctx}
}
