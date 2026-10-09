package shadow

const (
	// DefaultIDMin is the lowest non-system ID, the shadow default for UID_MIN and GID_MIN.
	DefaultIDMin = 1000
	// DefaultIDMax is the highest non-system ID, the shadow default for UID_MAX and GID_MAX.
	DefaultIDMax = 60000
)

// IsSystemID reports whether id is outside the non-system range.
// Nil idMin and idMax default to [DefaultIDMin] and [DefaultIDMax].
func IsSystemID(id int, idMin, idMax *int) bool {
	lower := DefaultIDMin
	if idMin != nil {
		lower = *idMin
	}
	upper := DefaultIDMax
	if idMax != nil {
		upper = *idMax
	}
	return id < lower || id > upper
}
