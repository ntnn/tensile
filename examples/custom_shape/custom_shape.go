package main

import (
	"context"
	"log"
	"time"

	"github.com/ntnn/tensile"
	"github.com/ntnn/tensile/engines"
	"golang.org/x/exp/slog"
)

const (
	// This is a custom shape that will be recognized by queues and
	// engines.
	// Custom shapes are useful for e.g. implementing custom resources
	// like uci for OpenWRT.
	MyShape tensile.Shape = "myshape"
)

// Two nodes utilizing the custom shape.
var _ tensile.Node = (*MyNodeA)(nil)

type MyNodeA struct {
	Message string
}

func (my MyNodeA) Identity() (tensile.Shape, string) {
	return MyShape, my.Message
}

func (my MyNodeA) Validate() error {
	return nil
}

func (my MyNodeA) Execute(ctx tensile.Context) (any, error) {
	ctx.Logger(my).Info(my.Message)
	return nil, nil
}

var _ tensile.Node = (*MyNodeB)(nil)

type MyNodeB struct {
	Message   string
	Timestamp time.Time
}

func (my MyNodeB) Identity() (tensile.Shape, string) {
	return MyShape, my.Message
}

func (my MyNodeB) Validate() error {
	return nil
}

func (my MyNodeB) Execute(ctx tensile.Context) (any, error) {
	ctx.Logger(my).Info(my.Message, slog.Time("timestamp", my.Timestamp))
	return nil, nil
}

func main() {
	tensile.SetDebugLog()
	if err := doMain(); err != nil {
		log.Fatal(err)
	}
}

func doMain() error {
	simple := engines.NewSimple(slog.Default())

	if err := simple.Queue.Add(
		&MyNodeA{
			Message: "Hello, world!",
		},
		&MyNodeB{
			// This would cause an error - MyNodeA and MyNodeB are
			// different nodes but both have the custom shape MyShape
			// and the same message, as such they would conflict.
			// Message: "Hello, world!",
			Message: "Hello, world 2!",
		},
	); err != nil {
		return err
	}

	return simple.Noop(context.Background())
}
