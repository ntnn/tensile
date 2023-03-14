package facts

func NewOSRelease() (OSRelease, error) {
	rel := OSRelease{}
	rel.Name = "windows"
	rel.ID = "windows"
	rel.IDLike = []string{"windows"}

	// rel.PrettyName = OS Name
	// rel.Version = OS Version / Version
	// rel.BuildID = OS Version / Build

	return rel, nil
}
