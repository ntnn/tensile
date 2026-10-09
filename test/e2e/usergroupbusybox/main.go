// usergroupbusybox exercises user and group management on busybox.
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

	uid := 1080
	gid := 2080

	group := &std.Group{
		Name: "e2e-busybox-group",
		GID:  &gid,
	}
	user := &std.User{
		Name:  "e2e-busybox-user",
		UID:   &uid,
		Group: "e2e-busybox-group",
		Home:  "/home/e2e-busybox-user",
		Shell: "/bin/sh",
	}
	member := &std.GroupMembership{User: "e2e-busybox-user", Group: "e2e-busybox-group"}

	q.Add(group, user, member)
	q.DependsOn(user, group.Identity())
	q.DependsOn(member, user.Identity(), group.Identity())

	return app.New().Run(ctx, q)
}
