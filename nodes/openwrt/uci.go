package openwrt

import "strings"

// UCIState is the desired presence of a UCI entity.
type UCIState string

// Desired presence states for UCI nodes.
const (
	UCIPresent UCIState = "present"
	UCIAbsent  UCIState = "absent"
)

// uciNotFound reports whether out is uci's missing-entry error.
func uciNotFound(out []byte) bool {
	return strings.Contains(string(out), "Entry not found")
}

// parseUCIValues parses uci show values into their string values.
//
// uci show renders option values single-quoted and space-separated:
//
//	'world'                     → [world]
//	'a' 'b'                     → [a b]           (list)
//	'other config'              → [other config]  (space inside quotes)
//	'it'\''s'                   → [it's]          (shell-escaped quote)
func parseUCIValues(s string) []string {
	var values []string
	var current strings.Builder
	started := false

	flush := func() {
		if started {
			values = append(values, current.String())
			current.Reset()
			started = false
		}
	}

	for i := 0; i < len(s); i++ {
		switch c := s[i]; c {
		case '\'':
			// quoted segment, e.g. 'world' or the 'it' and 's' parts
			// of 'it'\''s'
			started = true
			i = consumeQuoted(s, i+1, &current)
		case '\\':
			// escaped quote between segments: the \' in 'it'\''s'
			if i+1 < len(s) && s[i+1] == '\'' {
				current.WriteByte('\'')
				started = true
				i++
				continue
			}
			current.WriteByte(c)
			started = true
		case ' ', '\t':
			// separator between values, e.g. between 'a' and 'b'
			flush()
		default:
			// unquoted byte, uci quotes everything but stay tolerant
			current.WriteByte(c)
			started = true
		}
	}
	flush()
	return values
}

// consumeQuoted appends bytes until the closing quote and returns its index.
func consumeQuoted(s string, start int, b *strings.Builder) int {
	for i := start; i < len(s); i++ {
		if s[i] == '\'' {
			return i
		}
		b.WriteByte(s[i])
	}
	return len(s)
}
