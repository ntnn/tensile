package std

import (
	"context"
	"slices"

	"github.com/ntnn/tensile/pkg/shadow"
)

var _ shadow.Service = (*fakeShadow)(nil)

// fakeShadow is an in-memory [shadow.Service].
type fakeShadow struct {
	users  []shadow.User
	groups []shadow.Group
	// nextID is the ID assigned when a spec leaves it unset.
	nextID int
	// removedHomes are the users deleted with removeHome.
	removedHomes []string
}

func (f *fakeShadow) User(name string) (*shadow.User, error) {
	for _, user := range f.users {
		if user.Name == name {
			return &user, nil
		}
	}
	return nil, nil //nolint:nilnil // nil means the user does not exist
}

func (f *fakeShadow) Group(name string) (*shadow.Group, error) {
	for _, group := range f.groups {
		if group.Name == name {
			return &group, nil
		}
	}
	return nil, nil //nolint:nilnil // nil means the group does not exist
}

func (f *fakeShadow) Users() ([]shadow.User, error) {
	return slices.Clone(f.users), nil
}

func (f *fakeShadow) Groups() ([]shadow.Group, error) {
	return slices.Clone(f.groups), nil
}

func (f *fakeShadow) ApplyUser(_ context.Context, desired shadow.UserSpec) error {
	i := slices.IndexFunc(f.users, func(u shadow.User) bool { return u.Name == desired.Name })
	if i < 0 {
		f.users = append(f.users, shadow.User{
			Name: desired.Name,
			UID:  f.id(desired.UID),
		})
		i = len(f.users) - 1
	}
	user := &f.users[i]
	if desired.UID != nil {
		user.UID = *desired.UID
	}
	if desired.Group != "" {
		user.Group = desired.Group
	}
	if desired.Groups != nil {
		user.Groups = slices.Clone(desired.Groups)
	}
	if desired.Home != "" {
		user.Home = desired.Home
	}
	if desired.Shell != "" {
		user.Shell = desired.Shell
	}
	return nil
}

func (f *fakeShadow) DeleteUser(_ context.Context, name string, removeHome bool) error {
	if removeHome {
		f.removedHomes = append(f.removedHomes, name)
	}
	f.users = slices.DeleteFunc(f.users, func(u shadow.User) bool { return u.Name == name })
	return nil
}

func (f *fakeShadow) ApplyGroup(_ context.Context, desired shadow.GroupSpec) error {
	i := slices.IndexFunc(f.groups, func(g shadow.Group) bool { return g.Name == desired.Name })
	if i < 0 {
		f.groups = append(f.groups, shadow.Group{
			Name: desired.Name,
			GID:  f.id(desired.GID),
		})
		return nil
	}
	if desired.GID != nil {
		f.groups[i].GID = *desired.GID
	}
	return nil
}

func (f *fakeShadow) DeleteGroup(_ context.Context, name string) error {
	f.groups = slices.DeleteFunc(f.groups, func(g shadow.Group) bool { return g.Name == name })
	return nil
}

func (f *fakeShadow) AddMember(_ context.Context, user, group string) error {
	for i := range f.groups {
		if f.groups[i].Name == group && !slices.Contains(f.groups[i].Members, user) {
			f.groups[i].Members = append(f.groups[i].Members, user)
		}
	}
	for i := range f.users {
		if f.users[i].Name == user && !slices.Contains(f.users[i].Groups, group) {
			f.users[i].Groups = append(f.users[i].Groups, group)
		}
	}
	return nil
}

func (f *fakeShadow) RemoveMember(_ context.Context, user, group string) error {
	for i := range f.groups {
		if f.groups[i].Name == group {
			f.groups[i].Members = slices.DeleteFunc(f.groups[i].Members, func(m string) bool { return m == user })
		}
	}
	for i := range f.users {
		if f.users[i].Name == user {
			f.users[i].Groups = slices.DeleteFunc(f.users[i].Groups, func(g string) bool { return g == group })
		}
	}
	return nil
}

// id returns the requested ID or allocates the next one.
func (f *fakeShadow) id(requested *int) int {
	if requested != nil {
		return *requested
	}
	f.nextID++
	return f.nextID
}
