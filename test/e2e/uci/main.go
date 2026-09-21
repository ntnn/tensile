// uci exercises uci nodes
package main

import (
	"context"
	"log"

	"github.com/ntnn/tensile"
	"github.com/ntnn/tensile/nodes/openwrt"
	"github.com/ntnn/tensile/pkg/engine"
	"github.com/ntnn/tensile/pkg/queue"
)

// Managed configs and sections, files seeded by the test.
const (
	config     = "tensile"
	section    = "main"
	srvSection = "srv"
	// config2 is committed only through the global handler.
	config2  = "tensile2"
	section2 = "main2"

	// value for the int option.
	portValue = 8080
)

func main() {
	if err := run(context.Background()); err != nil {
		log.Fatal(err)
	}
}

//nolint:funlen // scenario enumerates the node combinations
func run(ctx context.Context) error {
	q := queue.New()

	// package-default style anonymous sections, seeded by the test
	wipe := &openwrt.UCIDeleteAnonymousSections{
		Config: config,
		Type:   "host",
	}

	// named host section, implicitly ordered after the wipe
	srv := &openwrt.UCISection{
		Config:  config,
		Section: srvSection,
		Type:    "host",
	}
	srvName := &openwrt.UCIOption[string]{
		Config:  config,
		Section: srvSection,
		Option:  "name",
		Value:   srvSection,
	}
	// seeded by the test beforehand, must be removed
	obsolete := &openwrt.UCISection{
		Config:  config,
		Section: "obsolete",
		State:   openwrt.UCIAbsent,
	}

	// one option per value type
	hello := &openwrt.UCIOption[string]{
		Config:  config,
		Section: section,
		Option:  "hello",
		Value:   "world",
	}
	port := &openwrt.UCIOption[int]{
		Config:  config,
		Section: section,
		Option:  "port",
		Value:   portValue,
	}
	ignore := &openwrt.UCIOption[bool]{
		Config:  config,
		Section: section,
		Option:  "ignore",
		Value:   true,
	}
	dns := &openwrt.UCIOption[[]string]{
		Config:  config,
		Section: section,
		Option:  "dns",
		Value:   []string{"192.168.178.5", "1.1.1.1"},
	}
	ports := &openwrt.UCIOption[[]int]{
		Config:  config,
		Section: section,
		Option:  "ports",
		Value:   []int{80, 443},
	}
	// seeded by the test beforehand, must be removed
	legacy := &openwrt.UCIOption[string]{
		Config:  config,
		Section: section,
		Option:  "legacy",
		State:   openwrt.UCIAbsent,
	}

	// second config, committed only by the global handler
	other := &openwrt.UCISection{
		Config:  config2,
		Section: section2,
		Type:    "settings",
	}
	greet := &openwrt.UCIOption[string]{
		Config:  config2,
		Section: section2,
		Option:  "greet",
		Value:   "hi",
	}

	// per-config commit for tensile, global commit for the rest
	commit := tensile.NewHandler(&openwrt.UCICommit{Config: config})
	commitAll := tensile.NewHandler(&openwrt.UCICommit{})

	if err := q.Enqueue(
		wipe, srv, srvName, obsolete,
		hello, port, ignore, dns, ports, legacy,
		other, greet,
		commit, commitAll,
	); err != nil {
		return err
	}
	if err := q.DependsOn(srvName, srv); err != nil {
		return err
	}
	if err := q.DependsOn(greet, other); err != nil {
		return err
	}

	work, err := q.Build()
	if err != nil {
		return err
	}

	seq := engine.NewParallel(work, engine.ParallelOptions{})
	return seq.Execute(ctx)
}
