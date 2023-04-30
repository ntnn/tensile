package nodes

func init() {
	RegisterPackageProvider("arch", PackageProviderPacman)
}

type PackageProviderPacman struct {}

func (pacman PackageProviderPacman) IsInstalled(state PackageState, package string) (bool, error) {
	return false, nil
}

func (pacman PackageProviderPacman) Install(state PackageState, package string) (bool, error) {
	return false, nil
}
