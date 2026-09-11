package std

import (
	"context"
)

// PackageManager queries and changes installed packages.
type PackageManager interface {
	// Handles reports whether this manager handles the package.
	Handles(ctx context.Context, name string) (bool, error)
	// Installed returns whether the package is installed.
	Installed(ctx context.Context, name string) (bool, error)
	// Install installs the package.
	Install(ctx context.Context, name string) error
	// Remove removes the package.
	Remove(ctx context.Context, name string) error
}

var packageManagers = registry[PackageManager]{kind: "package manager"}

// RegisterPackageManager registers a package manager.
// Panics when name is already registered.
func RegisterPackageManager(name string, manager PackageManager) {
	packageManagers.register(name, manager)
}

// PackageManagerByName returns the package manager registered under name.
func PackageManagerByName(name string) (PackageManager, error) {
	return packageManagers.byName(name)
}

// DetectPackageManager returns the last registered package manager
// that handles the named package.
func DetectPackageManager(ctx context.Context, name string) (PackageManager, error) {
	return packageManagers.detect(ctx, name)
}
