package facts

// OSRelease is filled from the os-release file where available. On
// hosts without an equivalent file a best effort is done to fill the
// same fields.
//
// Linux data is read from os-release if available.
// Windows data is taken from systeminfo.exe.
type OSRelease struct {
	Name       string `osrel:"NAME" json:"name"`
	PrettyName string `osrel:"PRETTY_NAME" json:"pretty_name"`

	CPEName string `osrel:"CPE_NAME" json:"cpe_name"`

	ID     string   `osrel:"ID" json:"id"`
	IDLike []string `osrel:"ID_LIKE" json:"id_like"`

	Variant   string `osrel:"VARIANT" json:"variant"`
	VariantID string `osrel:"VARIANT_ID" json:"variant_id"`

	Version         string `osrel:"VERSION" json:"version"`
	VersionID       string `osrel:"VERSION_ID" json:"version_id"`
	VersionCodename string `osrel:"VERSION_CODENAME" json:"version_codename"`

	BuildID string `osrel:"BUILD_ID" json:"build_id"`

	ImageID      string `osrel:"IMAGE_ID" json:"image_id"`
	ImageVersion string `osrel:"IMAGE_VERSION" json:"image_version"`
}
