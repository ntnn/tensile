package main

import (
	"encoding/json"
	"log"
	"os"

	"github.com/ntnn/tensile/facts"
)

func main() {
	if err := doMain(); err != nil {
		log.Fatal(err)
	}
}

func doMain() error {
	f, err := facts.New()
	if err != nil {
		return err
	}

	encoder := json.NewEncoder(os.Stdout)
	encoder.SetIndent("  ", "  ")

	if err := encoder.Encode(f); err != nil {
		return err
	}

	return nil
}
