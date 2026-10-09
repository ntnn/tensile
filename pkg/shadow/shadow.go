package shadow

import (
	"context"
	"errors"
	"fmt"
	"os/exec"
	"path/filepath"
)

// User is a local user.
type User struct {
	Name string
	UID  int
	GID  int
	// Group is the primary group name, empty when GID resolves to no group.
	Group string
	// Groups are the supplementary groups listing the user as a member.
	Groups []string
	Home   string
	Shell  string
}

// Group is a local group.
type Group struct {
	Name string
	GID  int
	// Members are the users listed as supplementary members.
	Members []string
}

// UserSpec is the desired state of a user.
// Unset fields are unmanaged.
type UserSpec struct {
	Name string
	UID  *int
	// Group empty applies the tool default on creation.
	Group string
	// Groups is the complete set of supplementary groups.
	// Nil is unmanaged, empty removes all supplementary groups.
	Groups []string
	// Home empty applies the tool default on creation.
	Home string
	// Shell empty applies the tool default on creation.
	Shell string
}

// GroupSpec is the desired state of a group.
// Unset fields are unmanaged.
type GroupSpec struct {
	Name string
	GID  *int
}

// Service reads and changes local users and groups.
// Unsupported operations return an error wrapping [errors.ErrUnsupported].
type Service interface {
	// User returns the named user, nil when it does not exist.
	User(name string) (*User, error)
	// Group returns the named group, nil when it does not exist.
	Group(name string) (*Group, error)
	// Users returns all users in file order.
	Users() ([]User, error)
	// Groups returns all groups in file order.
	Groups() ([]Group, error)
	// ApplyUser creates or modifies the user.
	ApplyUser(ctx context.Context, desired UserSpec) error
	// DeleteUser deletes the user.
	// removeHome also removes the home. Note that home removal is
	// governed by the backing tool and not by the package.
	DeleteUser(ctx context.Context, name string, removeHome bool) error
	// ApplyGroup creates or modifies the group.
	ApplyGroup(ctx context.Context, desired GroupSpec) error
	// DeleteGroup deletes the group.
	DeleteGroup(ctx context.Context, name string) error
	// AddMember adds the user as supplementary member of the group.
	AddMember(ctx context.Context, user, group string) error
	// RemoveMember removes the user as supplementary member of the group.
	RemoveMember(ctx context.Context, user, group string) error
}

// Options configures the [Service].
type Options struct {
	// Root is the directory containing etc/passwd and etc/group, e.g. a jail.
	// Empty means "/".
	// Busybox does not support a root other than "/".
	Root string
}

// New returns the Service for the available tools.
// The shadow tools win over busybox when both are installed.
func New(opts Options) (Service, error) {
	root := opts.Root
	if root == "" {
		root = "/"
	}
	if !filepath.IsAbs(root) {
		return nil, fmt.Errorf("root %q is not absolute", opts.Root)
	}
	root = filepath.Clean(root)

	f := files{
		passwdPath: filepath.Join(root, "etc", "passwd"),
		groupPath:  filepath.Join(root, "etc", "group"),
	}
	if _, err := exec.LookPath("useradd"); err == nil {
		return &tools{
			files: f,
			root:  root,
		}, nil
	}
	if _, err := exec.LookPath("adduser"); err == nil {
		if root != "/" {
			return nil, fmt.Errorf("busybox with root %q: %w", root, errors.ErrUnsupported)
		}
		return &busybox{files: f}, nil
	}
	return nil, errors.New("neither the shadow tools (useradd) nor busybox (adduser) are installed")
}
