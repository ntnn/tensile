// usergroupmodify exercises user and group modifications, pruning, and conflicts.
package main

import (
	"context"
	"log"
	"os"

	"github.com/ntnn/tensile/nodes/std"
	"github.com/ntnn/tensile/pkg/app"
	"github.com/ntnn/tensile/pkg/queue"
)

const (
	uid1         = 1060
	uid2         = 1061
	uidWheelMgmt = 1075
	gid1         = 2060
	gid2         = 2061
)

func main() {
	if err := run(context.Background()); err != nil {
		log.Fatal(err)
	}
}

func run(ctx context.Context) error {
	phase := "1"
	if len(os.Args) > 1 {
		phase = os.Args[1]
	}

	switch phase {
	case "create":
		return phaseCreate(ctx)
	case "prune":
		return phasePrune(ctx)
	case "modify":
		return phaseModify(ctx)
	case "remove":
		return phaseRemove(ctx)
	case "empty-groups":
		return phaseEmptyGroups(ctx)
	case "conflict":
		return phaseConflict()
	case "prune-wheel":
		return phasePruneWheel(ctx)
	default:
		return nil
	}
}

func managedEntities(gid, shell1 int, _ int) *queue.Queue {
	q := queue.New()

	team := &std.Group{Name: "e2e-team", GID: new(gid)}
	managed1 := &std.User{Name: "e2e-managed1", UID: new(uid1), Shell: "/bin/bash"}
	managed2 := &std.User{Name: "e2e-managed2", UID: new(uid2), Shell: "/bin/bash"}
	if shell1 != 0 {
		managed1.Shell = "/bin/sh"
	}
	m1 := &std.GroupMembership{User: "e2e-managed1", Group: "e2e-team"}
	m2 := &std.GroupMembership{User: "e2e-managed2", Group: "e2e-team"}

	q.Add(team, managed1, managed2, m1, m2)
	q.DependsOn(managed1, team.Identity())
	q.DependsOn(managed2, team.Identity())
	q.DependsOn(m1, managed1.Identity(), team.Identity())
	q.DependsOn(m2, managed2.Identity(), team.Identity())

	return q
}

// phaseCreate creates managed users, groups, and memberships.
func phaseCreate(ctx context.Context) error {
	return app.New().Run(ctx, managedEntities(gid1, 0, 0))
}

// phasePrune adds pruning nodes and runs them.
func phasePrune(ctx context.Context) error {
	q := managedEntities(gid1, 0, 0)
	q.Add(&std.PruneUnmanagedUsers{}, &std.PruneUnmanagedGroups{}, &std.PruneUnmanagedGroupMemberships{})
	return app.New().Run(ctx, q)
}

// phaseModify changes user shell and group GID.
func phaseModify(ctx context.Context) error {
	return app.New().Run(ctx, managedEntities(gid2, 1, 0))
}

// phaseRemove removes managed2 declarations and prunes again.
func phaseRemove(ctx context.Context) error {
	q := queue.New()

	team := &std.Group{Name: "e2e-team", GID: new(gid2)}
	managed1 := &std.User{Name: "e2e-managed1", UID: new(uid1), Shell: "/bin/sh"}
	m1 := &std.GroupMembership{User: "e2e-managed1", Group: "e2e-team"}

	pruneUsers := &std.PruneUnmanagedUsers{}
	pruneMembers := &std.PruneUnmanagedGroupMemberships{}

	q.Add(team, managed1, m1, pruneUsers, pruneMembers)
	q.DependsOn(managed1, team.Identity())
	q.DependsOn(m1, managed1.Identity(), team.Identity())

	return app.New().Run(ctx, q)
}

// phaseEmptyGroups tests empty groups removes all supplementary.
func phaseEmptyGroups(ctx context.Context) error {
	q := queue.New()

	emptyGroups := &std.User{
		Name:   "e2e-managed1",
		UID:    new(uid1),
		Shell:  "/bin/sh",
		Groups: []string{},
	}

	q.Add(emptyGroups)
	return app.New().Run(ctx, q)
}

// phaseConflict declares both User.Groups and GroupMembership for the same pair.
func phaseConflict() error {
	q := queue.New()
	q.Add(
		&std.User{Name: "e2e-conflict", Groups: []string{"e2e-team"}},
		&std.GroupMembership{User: "e2e-conflict", Group: "e2e-team"},
	)
	_, err := q.Build()
	if err != nil {
		log.Println("conflict detected:", err)
		return nil
	}
	return err
}

// phasePruneWheel prunes unmanaged wheel membership, keeps managed one.
func phasePruneWheel(ctx context.Context) error {
	q := queue.New()

	// managed wheel membership
	managedUser := &std.User{Name: "e2e-wheel-managed", UID: new(uidWheelMgmt)}
	managed := &std.GroupMembership{
		User:  "e2e-wheel-managed",
		Group: "wheel",
	}
	// unmanaged wheel membership should be pruned
	pruneWheel := &std.PruneUnmanagedGroupMemberships{
		Group:         "wheel",
		IncludeSystem: true,
	}

	q.Add(managedUser, managed, pruneWheel)
	q.DependsOn(managed, managedUser.Identity())

	return app.New().Run(ctx, q)
}
