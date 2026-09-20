package openwrt

import "strings"

// uciNotFound reports whether out is uci's missing-entry error.
func uciNotFound(out []byte) bool {
	return strings.Contains(string(out), "Entry not found")
}
