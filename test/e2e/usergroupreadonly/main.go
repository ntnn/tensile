// usergroupreadonly exercises ReadOnly state on users, groups, and memberships.
package main

import (
	"context"
	"log"
	"os"

	"github.com/ntnn/tensile"
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
	phase := "create"
	if len(os.Args) > 1 {
		phase = os.Args[1]
	}

	switch phase {
	case "create":
		return phaseCreate(ctx)
	case "readonly":
		return phaseReadOnly(ctx)
	case "readonly-membership":
		return phaseReadOnlyMembership(ctx)
	default:
		return nil
	}
}

// phaseCreate creates a user, group, and membership.
func phaseCreate(ctx context.Context) error {
	q := queue.New()

	uid := 1070
	gid := 2070

	group := &std.Group{
		Name: "e2e-readonly-group",
		GID:  &gid,
	}
	user := &std.User{
		Name:  "e2e-readonly-user",
		UID:   &uid,
		Group: "e2e-readonly-group",
		Shell: "/bin/bash",
	}
	member := &std.GroupMembership{User: "e2e-readonly-user", Group: "e2e-readonly-group"}

	q.Add(group, user, member)
	q.DependsOn(user, group.Identity())
	q.DependsOn(member, user.Identity(), group.Identity())

	return app.New().Run(ctx, q)
}

// phaseReadOnly sets all to ReadOnly with drifted values; no changes should apply.
func phaseReadOnly(ctx context.Context) error {
	q := queue.New()

	uid := 1071
	gid := 2071

	// drifted GID: should not be changed because ReadOnly
	group := &std.Group{
		Name:  "e2e-readonly-group",
		GID:   &gid,
		State: tensile.ReadOnly,
	}
	// drifted shell: should not be changed because ReadOnly
	user := &std.User{
		Name:  "e2e-readonly-user",
		UID:   &uid,
		Shell: "/bin/sh",
		State: tensile.ReadOnly,
	}

	q.Add(group, user)
	q.DependsOn(user, group.Identity())

	return app.New().Run(ctx, q)
}

// phaseReadOnlyMembership tries ReadOnly on GroupMembership; should fail validation at execution.
func phaseReadOnlyMembership(ctx context.Context) error {
	q := queue.New()
	m := &std.GroupMembership{
		User:  "e2e-readonly-user",
		Group: "e2e-readonly-group",
		State: tensile.ReadOnly,
	}
	q.Add(m)
	return app.New().Run(ctx, q)
}
