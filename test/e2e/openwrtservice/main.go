// openwrtservice enables and starts a service.
package main

import (
	"context"
	"log"

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
	q := queue.New()

	// cron refuses to start with an empty /etc/crontabs
	crontab := &std.FileContent{
		Path:    "/etc/crontabs/root",
		Content: "* * * * * true\n",
	}
	enabled, running := true, true
	service := &std.Service{
		Name:    "cron",
		Enabled: &enabled,
		Running: &running,
	}

	if err := q.Enqueue(crontab, service); err != nil {
		return err
	}
	if err := q.DependsOn(service, crontab); err != nil {
		return err
	}

	work, err := q.Build()
	if err != nil {
		return err
	}

	seq := engine.NewParallel(work, engine.ParallelOptions{})
	return seq.Execute(ctx)
}
