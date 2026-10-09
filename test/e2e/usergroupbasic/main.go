// usergroupbasic creates a user, a group, and a supplementary membership.
package main

import (
	"context"
	"log"

	"github.com/ntnn/tensile/nodes/std"
	"github.com/ntnn/tensile/pkg/app"
	"github.com/ntnn/tensile/pkg/queue"
)

func main() {
	if err := run(context.Background()); err != nil {
		log.Fatal(err)
	}
}

func run(ctx context.Context) error {
	q := queue.New()

	uid := 1050
	gid := 2050
	usersGroup := &std.Group{
		Name: "e2e-users",
		GID:  &gid,
	}
	wheelGroup := &std.Group{
		Name: "e2e-wheel",
	}
	alice := &std.User{
		Name:   "e2e-alice",
		UID:    &uid,
		Group:  "e2e-users",
		Home:   "/home/e2e-alice",
		Shell:  "/bin/bash",
		Groups: []string{"e2e-wheel"},
	}

	q.Add(usersGroup, wheelGroup, alice)
	q.DependsOn(alice, usersGroup.Identity(), wheelGroup.Identity())

	return app.New().Run(ctx, q)
}
