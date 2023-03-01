package main

import (
	"context"
	"log"

	"github.com/ntnn/gorrect"
	"github.com/ntnn/gorrect/engines"
	"golang.org/x/exp/slog"
)

const (
	MyIdentity gorrect.Identity = "myidentity"
)

type MyElement struct {
	Message string
}

func (my MyElement) Identity() (gorrect.Identity, string) {
	return MyIdentity, my.Message
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

	if err := simple.Add(
		&MyElement{
			Message: "Hello, world!",
		},
	); err != nil {
		return err
	}

	return simple.Run(context.Background())
}
