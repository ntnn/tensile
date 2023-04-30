package nodes

type PackageProvider interface {
	IsInstalled(state PackageState, package string) (bool, error)
	Install(state PackageState, package string) (bool, error)
}

var packageProviders = map[string]PackageProvider{}

func RegisterPackageProvider(osid string, provider PackageProvider) {
	packageProviders[osid] = provider
}

func ChoosePackageProvider(osid string) (PackageProvider, bool) {
	return packageProviders[osid]
}
