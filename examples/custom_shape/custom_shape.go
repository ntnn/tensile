package main

import (
	"context"
	"log"

	"github.com/ntnn/tensile"
	"github.com/ntnn/tensile/engines"
	"golang.org/x/exp/slog"
)

const (
	MyShape tensile.Shape = "myshape"
)

type MyElement struct {
	Message string
}

func (my MyElement) Identity() (tensile.Shape, string) {
	return MyShape, my.Message
}

func (my MyElement) Execute(ctx context.Context) error {
	log.Println(my.Message)
	return nil
}

func main() {
	if err := doMain(); err != nil {
		log.Fatal(err)
	}
}

func doMain() error {
	simple := engines.NewSimple(slog.Default())

	if err := simple.Queue.Add(
		&MyElement{
			Message: "Hello, world!",
		},
	); err != nil {
		return err
	}

	return simple.Run(context.Background())
}
