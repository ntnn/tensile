// servicebasic deploys a file, a symlink and a dummy systemd service.
package main

import (
	"context"
	"log"

	"github.com/ntnn/tensile/nodes/std"
	"github.com/ntnn/tensile/pkg/engine"
	"github.com/ntnn/tensile/pkg/queue"
)

const unit = `[Unit]
Description=tensile e2e dummy service

[Service]
ExecStart=/bin/sleep infinity

[Install]
WantedBy=multi-user.target
`

func main() {
	if err := run(context.Background()); err != nil {
		log.Fatal(err)
	}
}

func run(ctx context.Context) error {
	q := queue.New()

	dir := &std.Dir{Path: "/opt/e2e"}
	hello := &std.FileContent{
		Path:    "/opt/e2e/hello.txt",
		Content: "hello from tensile\n",
	}
	link := &std.Symlink{
		Path:   "/opt/e2e/link",
		Target: "/opt/e2e/hello.txt",
	}
	unitFile := &std.FileContent{
		Path:    "/etc/systemd/system/e2e-dummy.service",
		Content: unit,
	}
	enabled, running := true, true
	service := &std.Service{
		Name:    "e2e-dummy.service",
		Enabled: &enabled,
		Running: &running,
	}

	q.Add(dir, hello, link, unitFile, service)
	q.DependsOn(service, unitFile)

	work, err := q.Build()
	if err != nil {
		return err
	}

	seq := engine.NewParallel(work, engine.ParallelOptions{})
	if err := seq.Execute(ctx); err != nil {
		return err
	}

	return nil
}
