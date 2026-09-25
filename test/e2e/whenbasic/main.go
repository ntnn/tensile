// whenbasic exercises tensile.When gating nodes on std.Facts output.
package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/ntnn/tensile"
	"github.com/ntnn/tensile/nodes/std"
	"github.com/ntnn/tensile/pkg/engine"
	"github.com/ntnn/tensile/pkg/queue"
)

func main() {
	if err := run(context.Background()); err != nil {
		log.Fatal(err)
	}
}

func run(ctx context.Context) error {
	if err := runGated(ctx); err != nil {
		return err
	}
	return runUndeclaredDep(ctx)
}

// factsOS reads the OS from the facts output.
func factsOS(wire tensile.Wire) (string, error) {
	data, err := wire.Storage().Get[std.FactsData](std.FactsIdentity())
	if err != nil {
		return "", err
	}
	return data.OS, nil
}

// runGated runs a queue with nodes gated on facts:
// a true condition executes, a false condition skips execution but
// the skipped node still reports for its dependers.
func runGated(ctx context.Context) error {
	q := queue.New()

	dir := &std.Dir{Path: "/opt/e2e"}
	facts := &std.Facts{}

	linuxCond := tensile.Cond(
		func(wire tensile.Wire) (bool, error) {
			os, err := factsOS(wire)
			return os == "linux", err
		},
		facts.Identity(),
	)
	plan9Cond := tensile.Cond(
		func(wire tensile.Wire) (bool, error) {
			os, err := factsOS(wire)
			return os == "plan9", err
		},
		facts.Identity(),
	)

	enabledFile := tensile.When(linuxCond, &std.File{Path: "/opt/e2e/when-true"})
	enabled := tensile.When(
		linuxCond,
		&std.FileContent{
			Path:    "/opt/e2e/when-true",
			Content: "enabled\n",
		},
	)

	disabledFile := tensile.When(plan9Cond, &std.File{Path: "/opt/e2e/when-false"})
	disabled := tensile.When(
		plan9Cond,
		&std.FileContent{
			Path:    "/opt/e2e/when-false",
			Content: "disabled\n",
		},
	)

	// reads the report of the disabled node
	consumer := &writeReport{
		Path: "/opt/e2e/when-false-report",
		Dep:  disabled.Identity(),
	}

	q.Add(dir, facts, enabledFile, enabled, disabledFile, disabled, consumer)

	work, err := q.Build()
	if err != nil {
		return err
	}

	if err := engine.NewParallel(work, engine.ParallelOptions{}).Execute(ctx); err != nil {
		return fmt.Errorf("gated queue failed: %w", err)
	}

	if _, err := os.Stat("/opt/e2e/when-false"); !os.IsNotExist(err) {
		return errors.New("disabled node must not create its file")
	}
	return nil
}

// runUndeclaredDep runs a queue where the condition reads the facts
// output without declaring the dependency, which must error.
func runUndeclaredDep(ctx context.Context) error {
	q := queue.New()

	facts := &std.Facts{}
	gated := tensile.When(
		// no dependency on facts
		tensile.Cond(
			func(wire tensile.Wire) (bool, error) {
				os, err := factsOS(wire)
				return os == "linux", err
			},
		),
		&std.FileContent{
			Path:    "/opt/e2e/when-undeclared",
			Content: "undeclared\n",
		},
	)

	q.Add(facts, gated)

	work, err := q.Build()
	if err != nil {
		return err
	}

	err = engine.NewParallel(work, engine.ParallelOptions{}).Execute(ctx)
	if err == nil {
		return errors.New("undeclared dependency read must fail")
	}
	if !strings.Contains(err.Error(), "not a dependency") {
		return fmt.Errorf("expected not-a-dependency error, got: %w", err)
	}
	if _, err := os.Stat("/opt/e2e/when-undeclared"); !os.IsNotExist(err) {
		return errors.New("failed node must not create its file")
	}
	return nil
}

// reportFileMode is the permission for written report artifacts.
const reportFileMode = 0o644

// writeReport writes the reported sha256 of a FileContent dependency to Path.
type writeReport struct {
	Path string
	Dep  tensile.Identity
}

var _ tensile.Identifier = (*writeReport)(nil)
var _ tensile.Depender = (*writeReport)(nil)
var _ tensile.Executor = (*writeReport)(nil)

func (w *writeReport) Identity() tensile.Identity {
	return tensile.AsIdentity("writeReport", "path", w.Path)
}

func (w *writeReport) DependsOn() ([]tensile.Identity, error) {
	return []tensile.Identity{w.Dep}, nil
}

func (w *writeReport) NeedsExecution(_ tensile.Wire) (bool, tensile.Diff, error) {
	return true, nil, nil
}

func (w *writeReport) Execute(wire tensile.Wire) (tensile.Diff, error) {
	out, err := wire.Storage().Get[std.FileContentOutput](w.Dep)
	if err != nil {
		return nil, fmt.Errorf("reading output of %s: %w", w.Dep, err)
	}
	if err := os.WriteFile(w.Path, []byte(out.SHA256), reportFileMode); err != nil {
		return nil, fmt.Errorf("writing report to %s: %w", w.Path, err)
	}
	return nil, nil //nolint:nilnil // nil Diff is valid
}
